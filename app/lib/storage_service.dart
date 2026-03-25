import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class SavedConnection {
  final String host;
  final int port;
  final String password;
  final String? fingerprint;

  SavedConnection({
    required this.host,
    required this.port,
    required this.password,
    this.fingerprint,
  });

  Map<String, dynamic> toJson() => {
        'host': host,
        'port': port,
        'password': password,
        'fingerprint': fingerprint,
      };

  factory SavedConnection.fromJson(Map<String, dynamic> json) {
    return SavedConnection(
      host: json['host'],
      port: json['port'],
      password: json['password'],
      fingerprint: json['fingerprint'],
    );
  }
}

class Macro {
  final String name;
  final String command;

  Macro({required this.name, required this.command});

  Map<String, dynamic> toJson() => {'name': name, 'command': command};
  factory Macro.fromJson(Map<String, dynamic> json) => Macro(name: json['name'], command: json['command']);
}

class ServerProfile {
  final String id;
  final String name;
  final String apiBaseUrl;
  final String oidcIssuer;
  final String clientId;
  final String redirectUrl;
  final List<String> scopes;
  final String? tlsFingerprint;

  const ServerProfile({
    required this.id,
    required this.name,
    required this.apiBaseUrl,
    required this.oidcIssuer,
    required this.clientId,
    required this.redirectUrl,
    required this.scopes,
    this.tlsFingerprint,
  });

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'apiBaseUrl': apiBaseUrl,
        'oidcIssuer': oidcIssuer,
        'clientId': clientId,
        'redirectUrl': redirectUrl,
        'scopes': scopes,
        'tlsFingerprint': tlsFingerprint,
      };

  factory ServerProfile.fromJson(Map<String, dynamic> json) {
    return ServerProfile(
      id: json['id'] as String,
      name: json['name'] as String,
      apiBaseUrl: json['apiBaseUrl'] as String,
      oidcIssuer: json['oidcIssuer'] as String,
      clientId: json['clientId'] as String,
      redirectUrl: json['redirectUrl'] as String,
      scopes: (json['scopes'] as List<dynamic>? ?? const <dynamic>[]).cast<String>(),
      tlsFingerprint: json['tlsFingerprint'] as String?,
    );
  }

  ServerProfile copyWith({
    String? id,
    String? name,
    String? apiBaseUrl,
    String? oidcIssuer,
    String? clientId,
    String? redirectUrl,
    List<String>? scopes,
    String? tlsFingerprint,
  }) {
    return ServerProfile(
      id: id ?? this.id,
      name: name ?? this.name,
      apiBaseUrl: apiBaseUrl ?? this.apiBaseUrl,
      oidcIssuer: oidcIssuer ?? this.oidcIssuer,
      clientId: clientId ?? this.clientId,
      redirectUrl: redirectUrl ?? this.redirectUrl,
      scopes: scopes ?? this.scopes,
      tlsFingerprint: tlsFingerprint ?? this.tlsFingerprint,
    );
  }
}

class PublicAuthSession {
  final String accessToken;
  final String? refreshToken;
  final String? idToken;
  final DateTime? accessTokenExpiration;

  const PublicAuthSession({
    required this.accessToken,
    this.refreshToken,
    this.idToken,
    this.accessTokenExpiration,
  });

  Map<String, dynamic> toJson() => {
        'accessToken': accessToken,
        'refreshToken': refreshToken,
        'idToken': idToken,
        'accessTokenExpiration': accessTokenExpiration?.toIso8601String(),
      };

  factory PublicAuthSession.fromJson(Map<String, dynamic> json) {
    return PublicAuthSession(
      accessToken: json['accessToken'] as String,
      refreshToken: json['refreshToken'] as String?,
      idToken: json['idToken'] as String?,
      accessTokenExpiration: json['accessTokenExpiration'] == null
          ? null
          : DateTime.tryParse(json['accessTokenExpiration'] as String),
    );
  }
}

class StorageService {
  static const _storage = FlutterSecureStorage();
  static const _connectionsKey = 'saved_connections';
  static const _macrosKey = 'saved_macros';
  static const _themeKey = 'selected_theme';
  static const _serverProfilesKey = 'server_profiles';
  static const _trustedFingerprintsKey = 'trusted_fingerprints';

  Future<Map<String, String>> getTrustedFingerprints() async {
    final String? data = await _storage.read(key: _trustedFingerprintsKey);
    if (data == null || data.isEmpty) return {};

    try {
      final Map<String, dynamic> decoded = jsonDecode(data);
      return decoded.cast<String, String>();
    } catch (e) {
      return {};
    }
  }

  Future<void> saveTrustedFingerprint(String host, String fingerprint) async {
    final fingerprints = await getTrustedFingerprints();
    fingerprints[host] = fingerprint;
    final String encoded = jsonEncode(fingerprints);
    await _storage.write(key: _trustedFingerprintsKey, value: encoded);
  }

  Future<String?> getTrustedFingerprint(String host) async {
    final fingerprints = await getTrustedFingerprints();
    return fingerprints[host];
  }

  Future<void> deleteTrustedFingerprint(String host) async {
    final fingerprints = await getTrustedFingerprints();
    fingerprints.remove(host);
    final String encoded = jsonEncode(fingerprints);
    await _storage.write(key: _trustedFingerprintsKey, value: encoded);
  }

  Future<List<SavedConnection>> getConnections() async {
    final String? data = await _storage.read(key: _connectionsKey);
    if (data == null || data.isEmpty) return [];

    try {
      final List<dynamic> decoded = jsonDecode(data);
      return decoded.map((e) => SavedConnection.fromJson(e)).toList();
    } catch (e) {
      return [];
    }
  }

  Future<void> saveConnection(SavedConnection newConn) async {
    final connections = await getConnections();
    // Remove existing entry for same host/port if it exists to update password
    connections.removeWhere((c) => c.host == newConn.host && c.port == newConn.port);
    connections.add(newConn);

    final String encoded = jsonEncode(connections.map((c) => c.toJson()).toList());
    await _storage.write(key: _connectionsKey, value: encoded);
  }

  Future<void> deleteConnection(String host, int port) async {
    final connections = await getConnections();
    connections.removeWhere((c) => c.host == host && c.port == port);
    final String encoded = jsonEncode(connections.map((c) => c.toJson()).toList());
    await _storage.write(key: _connectionsKey, value: encoded);
  }

  Future<String?> getPasswordFor(String host, int port) async {
    final connections = await getConnections();
    try {
      return connections.firstWhere((c) => c.host == host && c.port == port).password;
    } catch (e) {
      return null;
    }
  }

  Future<List<Macro>> getMacros() async {
    final String? data = await _storage.read(key: _macrosKey);
    if (data == null || data.isEmpty) {
      // Default macros
      return [
        Macro(name: 'ls', command: 'ls -la'),
        Macro(name: 'git status', command: 'git status'),
        Macro(name: 'top', command: 'top'),
        Macro(name: 'clear', command: 'clear'),
      ];
    }

    try {
      final List<dynamic> decoded = jsonDecode(data);
      return decoded.map((e) => Macro.fromJson(e)).toList();
    } catch (e) {
      return [];
    }
  }

  Future<void> saveMacros(List<Macro> macros) async {
    final String encoded = jsonEncode(macros.map((m) => m.toJson()).toList());
    await _storage.write(key: _macrosKey, value: encoded);
  }

  Future<String> getSelectedThemeName() async {
    return await _storage.read(key: _themeKey) ?? 'Dracula';
  }

  Future<void> saveSelectedThemeName(String name) async {
    await _storage.write(key: _themeKey, value: name);
  }

  Future<List<ServerProfile>> getServerProfiles() async {
    final String? data = await _storage.read(key: _serverProfilesKey);
    if (data == null || data.isEmpty) return [];

    try {
      final List<dynamic> decoded = jsonDecode(data);
      return decoded.map((e) => ServerProfile.fromJson(e as Map<String, dynamic>)).toList();
    } catch (e) {
      return [];
    }
  }

  Future<void> saveServerProfile(ServerProfile profile) async {
    final profiles = await getServerProfiles();
    final updated = profiles.where((item) => item.id != profile.id).toList()..add(profile);
    final String encoded = jsonEncode(updated.map((item) => item.toJson()).toList());
    await _storage.write(key: _serverProfilesKey, value: encoded);
  }

  Future<void> deleteServerProfile(String profileId) async {
    final profiles = await getServerProfiles();
    profiles.removeWhere((item) => item.id == profileId);
    final String encoded = jsonEncode(profiles.map((item) => item.toJson()).toList());
    await _storage.write(key: _serverProfilesKey, value: encoded);
    await clearPublicAuthSession(profileId);
  }

  Future<PublicAuthSession?> getPublicAuthSession(String profileId) async {
    final data = await _storage.read(key: _publicAuthSessionKey(profileId));
    if (data == null || data.isEmpty) return null;

    try {
      return PublicAuthSession.fromJson(jsonDecode(data) as Map<String, dynamic>);
    } catch (e) {
      return null;
    }
  }

  Future<void> savePublicAuthSession(String profileId, PublicAuthSession session) async {
    final encoded = jsonEncode(session.toJson());
    await _storage.write(key: _publicAuthSessionKey(profileId), value: encoded);
  }

  Future<void> clearPublicAuthSession(String profileId) async {
    await _storage.delete(key: _publicAuthSessionKey(profileId));
  }

  String _publicAuthSessionKey(String profileId) => 'public_auth_session_$profileId';
}
