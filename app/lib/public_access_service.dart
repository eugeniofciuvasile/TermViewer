import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:http/http.dart' as http;
import 'package:http/io_client.dart';

import 'storage_service.dart';

const String defaultRedirectUrl = 'com.termviewer.termviewer:/oauthredirect';
const List<String> defaultOidcScopes = <String>[
  'openid',
  'profile',
  'email',
  'offline_access',
];

class RemoteShareSession {
  final int id;
  final String status;
  final DateTime expiresAt;

  const RemoteShareSession({
    required this.id,
    required this.status,
    required this.expiresAt,
  });

  factory RemoteShareSession.fromJson(Map<String, dynamic> json) {
    return RemoteShareSession(
      id: json['id'] as int,
      status: json['status'] as String,
      expiresAt: DateTime.parse(json['expires_at'] as String),
    );
  }
}

class RemoteMachine {
  final int id;
  final String name;
  final String clientId;
  final String status;
  final DateTime? lastSeenAt;
  final RemoteShareSession? activeShareSession;

  const RemoteMachine({
    required this.id,
    required this.name,
    required this.clientId,
    required this.status,
    this.lastSeenAt,
    this.activeShareSession,
  });

  bool get canConnect => status == 'online' || status == 'waiting';

  factory RemoteMachine.fromJson(Map<String, dynamic> json) {
    return RemoteMachine(
      id: json['id'] as int,
      name: json['name'] as String,
      clientId: json['client_id'] as String,
      status: json['status'] as String,
      lastSeenAt: json['last_seen_at'] == null
          ? null
          : DateTime.tryParse(json['last_seen_at'] as String),
      activeShareSession: json['active_share_session'] == null
          ? null
          : RemoteShareSession.fromJson(
              json['active_share_session'] as Map<String, dynamic>,
            ),
    );
  }
}

class CreatedShareSession {
  final int sessionId;
  final String sessionToken;
  final String refreshToken;
  final DateTime expiresAt;
  final String serverUrl;
  final String deepLink;
  final String status;

  const CreatedShareSession({
    required this.sessionId,
    required this.sessionToken,
    required this.refreshToken,
    required this.expiresAt,
    required this.serverUrl,
    required this.deepLink,
    required this.status,
  });

  factory CreatedShareSession.fromJson(Map<String, dynamic> json) {
    return CreatedShareSession(
      sessionId: json['session_id'] as int,
      sessionToken: json['session_token'] as String,
      refreshToken: json['refresh_token'] as String,
      expiresAt: DateTime.parse(json['expires_at'] as String),
      serverUrl: json['server_url'] as String,
      deepLink: json['deep_link'] as String,
      status: json['status'] as String,
    );
  }
}

class ShareConnectResponse {
  final int sessionId;
  final String relayUrl;
  final String status;
  final DateTime expiresAt;
  final String? serverTlsFingerprint;

  const ShareConnectResponse({
    required this.sessionId,
    required this.relayUrl,
    required this.status,
    required this.expiresAt,
    this.serverTlsFingerprint,
  });

  factory ShareConnectResponse.fromJson(Map<String, dynamic> json) {
    return ShareConnectResponse(
      sessionId: json['session_id'] as int,
      relayUrl: json['relay_url'] as String,
      status: json['status'] as String,
      expiresAt: DateTime.parse(json['expires_at'] as String),
      serverTlsFingerprint: json['server_tls_fingerprint'] as String?,
    );
  }
}

class ShareDeepLink {
  final String serverUrl;
  final String sessionToken;
  final String? refreshToken;

  const ShareDeepLink({
    required this.serverUrl,
    required this.sessionToken,
    this.refreshToken,
  });
}

class PublicAccessService {
  PublicAccessService({http.Client? httpClient})
    : _httpClient = httpClient ?? http.Client();

  final http.Client _httpClient;

  http.Client _createSecureClient(String? trustedFingerprint) {
    if (trustedFingerprint == null || trustedFingerprint.isEmpty) {
      return _httpClient;
    }

    final httpClient = HttpClient()
      ..badCertificateCallback = (cert, host, port) {
        final hash = sha256.convert(cert.der);
        final fingerprint = hash.toString().toLowerCase().replaceAll(':', '');
        final expected = trustedFingerprint.toLowerCase().replaceAll(':', '');
        return fingerprint == expected;
      };

    return IOClient(httpClient);
  }

  void dispose() {
    _httpClient.close();
  }

  /// Probes a URL to extract its TLS certificate fingerprint.
  /// This is used for TOFU (Trust On First Use) when a certificate is self-signed.
  Future<String> probeFingerprint(String url) async {
    final uri = Uri.parse(url);
    final host = uri.host;

    String? probedFingerprint;

    final client = HttpClient();
    client.badCertificateCallback = (cert, host, port) {
      final hash = sha256.convert(cert.der);
      probedFingerprint = hash.toString().toLowerCase().replaceAll(':', '');
      return true; // We accept it just to finish the handshake and get the fingerprint
    };

    try {
      final request = await client.getUrl(uri).timeout(const Duration(seconds: 5));
      final response = await request.close();
      await response.drain();
    } catch (_) {
      // It might still fail after handshake, but if we got the fingerprint, we're good
    } finally {
      client.close();
    }

    if (probedFingerprint == null) {
      throw Exception('Could not extract TLS fingerprint from $host');
    }

    return probedFingerprint!;
  }

  Future<Map<String, dynamic>> getOidcConfig(
    String issuer,
    String? trustedFingerprint,
  ) async {
    final client = _createSecureClient(trustedFingerprint);
    final discoveryUrl = issuer.endsWith('/')
        ? '$issuer.well-known/openid-configuration'
        : '$issuer/.well-known/openid-configuration';

    final response = await client.get(Uri.parse(discoveryUrl));
    if (response.statusCode >= 400) {
      throw Exception(
        'Failed to fetch OIDC discovery document: ${response.statusCode}',
      );
    }

    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  Future<PublicAuthSession> signIn(
    ServerProfile profile,
    String username,
    String password,
  ) async {
    final config = await getOidcConfig(
      profile.oidcIssuer,
      profile.tlsFingerprint,
    );
    final tokenEndpoint = config['token_endpoint'] as String;

    final client = _createSecureClient(profile.tlsFingerprint);
    final scopesStr = profile.scopes.isEmpty
        ? defaultOidcScopes.join(' ')
        : profile.scopes.join(' ');

    final response = await client.post(
      Uri.parse(tokenEndpoint),
      headers: {'Content-Type': 'application/x-www-form-urlencoded'},
      body: {
        'grant_type': 'password',
        'client_id': profile.clientId,
        'username': username,
        'password': password,
        'scope': scopesStr,
      },
    );

    if (response.statusCode >= 400) {
      throw Exception('Login failed: ${response.statusCode} - ${response.body}');
    }

    final data = jsonDecode(response.body) as Map<String, dynamic>;

    return PublicAuthSession(
      accessToken: data['access_token'] as String,
      refreshToken: data['refresh_token'] as String?,
      idToken: data['id_token'] as String?,
      accessTokenExpiration: data['expires_in'] != null
          ? DateTime.now().add(Duration(seconds: data['expires_in'] as int))
          : null,
    );
  }

  Future<PublicAuthSession> refreshSession(
    ServerProfile profile,
    PublicAuthSession currentSession,
  ) async {
    final refreshToken = currentSession.refreshToken;
    if (refreshToken == null || refreshToken.isEmpty) {
      throw Exception('No refresh token available. Please sign in again.');
    }

    final config = await getOidcConfig(
      profile.oidcIssuer,
      profile.tlsFingerprint,
    );
    final tokenEndpoint = config['token_endpoint'] as String;

    final client = _createSecureClient(profile.tlsFingerprint);
    final scopesStr = profile.scopes.isEmpty
        ? defaultOidcScopes.join(' ')
        : profile.scopes.join(' ');

    final response = await client.post(
      Uri.parse(tokenEndpoint),
      headers: {'Content-Type': 'application/x-www-form-urlencoded'},
      body: {
        'grant_type': 'refresh_token',
        'client_id': profile.clientId,
        'refresh_token': refreshToken,
        'scope': scopesStr,
      },
    );

    if (response.statusCode >= 400) {
      throw Exception('Token refresh failed: ${response.statusCode} - ${response.body}');
    }

    final data = jsonDecode(response.body) as Map<String, dynamic>;

    return PublicAuthSession(
      accessToken: data['access_token'] as String,
      refreshToken: data['refresh_token'] as String? ?? refreshToken,
      idToken: data['id_token'] as String? ?? currentSession.idToken,
      accessTokenExpiration: data['expires_in'] != null
          ? DateTime.now().add(Duration(seconds: data['expires_in'] as int))
          : null,
    );
  }

  Future<PublicAuthSession> ensureValidSession(
    StorageService storage,
    ServerProfile profile, {
    bool forceLogin = false,
  }) async {
    if (forceLogin) {
      throw Exception('Login required. Please sign in again.');
    }

    final stored = await storage.getPublicAuthSession(profile.id);
    if (stored == null) {
      throw Exception('No session found. Please sign in.');
    }

    final expiresAt = stored.accessTokenExpiration;
    if (expiresAt == null ||
        DateTime.now().isBefore(
          expiresAt.subtract(const Duration(minutes: 1)),
        )) {
      return stored;
    }

    final refreshed = await refreshSession(profile, stored);
    await storage.savePublicAuthSession(profile.id, refreshed);
    return refreshed;
  }

  Future<List<RemoteMachine>> fetchMachines(
    ServerProfile profile,
    String accessToken, {
    String? trustedFingerprint,
  }) async {
    final client = _createSecureClient(trustedFingerprint);
    final response = await client.get(
      Uri.parse('${normalizeBaseUrl(profile.apiBaseUrl)}/api/machines'),
      headers: <String, String>{'Authorization': 'Bearer $accessToken'},
    );

    if (response.statusCode >= 400) {
      throw Exception('Failed to fetch machines: ${response.statusCode}');
    }

    final decoded = jsonDecode(response.body) as List<dynamic>;
    return decoded
        .map((item) => RemoteMachine.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<CreatedShareSession> createShareSession(
    ServerProfile profile,
    String accessToken,
    int machineId, {
    String? trustedFingerprint,
  }) async {
    final client = _createSecureClient(trustedFingerprint);
    final response = await client.post(
      Uri.parse(
        '${normalizeBaseUrl(profile.apiBaseUrl)}/api/machines/$machineId/share-session',
      ),
      headers: <String, String>{
        'Authorization': 'Bearer $accessToken',
        'Content-Type': 'application/json',
      },
      body: jsonEncode(const <String, dynamic>{}),
    );

    if (response.statusCode >= 400) {
      throw Exception(
        _extractError(
          response.body,
          fallback: 'Failed to create share session.',
        ),
      );
    }

    return CreatedShareSession.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Future<ShareConnectResponse> connectShareSession(
    String serverUrl,
    String sessionToken, {
    String? trustedFingerprint,
  }) async {
    final client = _createSecureClient(trustedFingerprint);
    final response = await client.post(
      Uri.parse('${normalizeBaseUrl(serverUrl)}/api/share-sessions/connect'),
      headers: const <String, String>{'Content-Type': 'application/json'},
      body: jsonEncode(<String, dynamic>{'session_token': sessionToken}),
    );

    if (response.statusCode >= 400) {
      throw Exception(
        _extractError(
          response.body,
          fallback: 'Failed to connect share session.',
        ),
      );
    }

    return ShareConnectResponse.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  ShareDeepLink parseShareDeepLink(String rawValue) {
    final uri = Uri.parse(rawValue.trim());
    if (uri.scheme != 'termviewer' || uri.host != 'connect') {
      throw FormatException('Unsupported QR format.');
    }

    final serverUrl = uri.queryParameters['server'];
    final sessionToken = uri.queryParameters['session_token'];
    final refreshToken = uri.queryParameters['refresh_token'];
    if (serverUrl == null ||
        serverUrl.isEmpty ||
        sessionToken == null ||
        sessionToken.isEmpty) {
      throw FormatException('QR code is missing the server or session token.');
    }

    return ShareDeepLink(
      serverUrl: normalizeBaseUrl(serverUrl),
      sessionToken: sessionToken,
      refreshToken: refreshToken == null || refreshToken.isEmpty
          ? sessionToken
          : refreshToken,
    );
  }

  String normalizeBaseUrl(String raw) {
    final trimmed = raw.trim().replaceAll(RegExp(r'/+$'), '');
    return trimmed;
  }

  String _extractError(String body, {required String fallback}) {
    try {
      final decoded = jsonDecode(body) as Map<String, dynamic>;
      return decoded['error'] as String? ?? fallback;
    } catch (_) {
      return fallback;
    }
  }
}
