import 'package:flutter/material.dart';
import '../public_access_service.dart';
import '../storage_service.dart';
import '../terminal_client.dart';
import 'qr_scanner_screen.dart';
import 'server_profile_editor_screen.dart';

typedef PublicConnectionCallback = Future<void> Function(TerminalClient client);

class RemoteAccessScreen extends StatefulWidget {
  const RemoteAccessScreen({super.key, required this.onConnected});

  final PublicConnectionCallback onConnected;

  @override
  State<RemoteAccessScreen> createState() => _RemoteAccessScreenState();
}

class _RemoteAccessScreenState extends State<RemoteAccessScreen> {
  final StorageService _storage = StorageService();
  final PublicAccessService _service = PublicAccessService();

  List<ServerProfile> _profiles = <ServerProfile>[];
  String? _selectedProfileId;
  PublicAuthSession? _authSession;
  List<RemoteMachine> _machines = <RemoteMachine>[];
  bool _loading = true;
  bool _busy = false;
  String? _error;

  ServerProfile? get _selectedProfile {
    if (_selectedProfileId == null) return null;
    for (final profile in _profiles) {
      if (profile.id == _selectedProfileId) {
        return profile;
      }
    }
    return null;
  }

  @override
  void initState() {
    super.initState();
    _loadProfiles();
  }

  @override
  void dispose() {
    _service.dispose();
    super.dispose();
  }

  Future<void> _loadProfiles() async {
    final profiles = await _storage.getServerProfiles();
    final selectedId =
        _selectedProfileId != null &&
            profiles.any((p) => p.id == _selectedProfileId)
        ? _selectedProfileId
        : profiles.isNotEmpty
        ? profiles.first.id
        : null;

    PublicAuthSession? authSession;
    if (selectedId != null) {
      authSession = await _storage.getPublicAuthSession(selectedId);
    }

    if (!mounted) return;
    setState(() {
      _profiles = profiles;
      _selectedProfileId = selectedId;
      _authSession = authSession;
      _machines = <RemoteMachine>[];
      _loading = false;
    });

    if (authSession != null) {
      await _refreshMachines();
    }
  }

  Future<void> _saveProfile(ServerProfile profile) async {
    await _storage.saveServerProfile(profile);
    await _loadProfiles();
  }

  Future<void> _openProfileEditor({ServerProfile? initialProfile}) async {
    final result = await Navigator.of(context).push<ServerProfile>(
      MaterialPageRoute(
        builder: (context) =>
            ServerProfileEditorScreen(initialProfile: initialProfile),
      ),
    );

    if (result != null) {
      await _saveProfile(result);
      if (!mounted) return;
      final authSession = await _storage.getPublicAuthSession(result.id);
      if (!mounted) return;
      setState(() {
        _selectedProfileId = result.id;
        _authSession = authSession;
      });
      if (authSession != null) {
        await _refreshMachines();
      }
    }
  }

  Future<void> _deleteSelectedProfile() async {
    final profile = _selectedProfile;
    if (profile == null) return;

    await _storage.deleteServerProfile(profile.id);
    await _loadProfiles();
  }

  Future<Map<String, String>?> _showLoginDialog() async {
    final usernameCtrl = TextEditingController();
    final passwordCtrl = TextEditingController();

    return showDialog<Map<String, String>>(
      context: context,
      barrierDismissible: false,
      builder: (context) => AlertDialog(
        title: const Text('Sign In'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: usernameCtrl,
              decoration: const InputDecoration(labelText: 'Username'),
              autofocus: true,
            ),
            const SizedBox(height: 8),
            TextField(
              controller: passwordCtrl,
              decoration: const InputDecoration(labelText: 'Password'),
              obscureText: true,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(null),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop({
              'username': usernameCtrl.text.trim(),
              'password': passwordCtrl.text,
            }),
            child: const Text('Login'),
          ),
        ],
      ),
    );
  }

  Future<void> _signIn({bool force = false}) async {
    final initialProfile = _selectedProfile;
    if (initialProfile == null) return;
    var profile = initialProfile;

    if (force || _authSession == null) {
      final credentials = await _showLoginDialog();
      if (credentials == null ||
          credentials['username']!.isEmpty ||
          credentials['password']!.isEmpty) {
        return; // Cancelled or empty
      }

      setState(() {
        _busy = true;
        _error = null;
      });

      try {
        if (profile.oidcIssuer.startsWith('https://') &&
            (profile.tlsFingerprint == null || profile.tlsFingerprint!.isEmpty)) {
          try {
            final fingerprint = await _service.probeFingerprint(profile.oidcIssuer);
            if (mounted) {
              final host = Uri.parse(profile.oidcIssuer).host;
              final accepted = await _showTofuDialog(host, fingerprint);
              if (accepted) {
                profile = profile.copyWith(tlsFingerprint: fingerprint);
                await _storage.saveServerProfile(profile);
                final updatedProfiles = await _storage.getServerProfiles();
                if (mounted) {
                  setState(() {
                    _profiles = updatedProfiles;
                  });
                }
              } else {
                throw Exception('User did not trust the certificate.');
              }
            }
          } catch (e) {
            debugPrint('OIDC fingerprint probe failed: $e');
          }
        }

        final session = await _service.signIn(
          profile,
          credentials['username']!,
          credentials['password']!,
        );
        await _storage.savePublicAuthSession(profile.id, session);

        if (!mounted) return;
        setState(() {
          _authSession = session;
        });
        await _refreshMachines();
      } catch (e) {
        if (!mounted) return;
        setState(() {
          _error = e.toString();
        });
      } finally {
        if (mounted) {
          setState(() {
            _busy = false;
          });
        }
      }
    } else {
      setState(() {
        _busy = true;
        _error = null;
      });
      try {
        final session = await _service.ensureValidSession(_storage, profile);
        await _storage.savePublicAuthSession(profile.id, session);
        if (!mounted) return;
        setState(() {
          _authSession = session;
        });
        await _refreshMachines();
      } catch (e) {
        if (!mounted) return;
        setState(() {
          _error = e.toString();
        });
      } finally {
        if (mounted) {
          setState(() {
            _busy = false;
          });
        }
      }
    }
  }

  Future<void> _signOut() async {
    final profile = _selectedProfile;
    if (profile == null) return;

    await _storage.clearPublicAuthSession(profile.id);
    if (!mounted) return;
    setState(() {
      _authSession = null;
      _machines = <RemoteMachine>[];
    });
  }

  Future<void> _refreshMachines() async {
    final profile = _selectedProfile;
    if (profile == null || _authSession == null) return;

    setState(() {
      _busy = true;
      _error = null;
    });

    try {
      final session = await _service.ensureValidSession(_storage, profile);
      await _storage.savePublicAuthSession(profile.id, session);
      
      final host = Uri.parse(profile.apiBaseUrl).host;
      final trustedFingerprint = await _storage.getTrustedFingerprint(host);

      final machines = await _service.fetchMachines(
        profile,
        session.accessToken,
        trustedFingerprint: trustedFingerprint,
      );

      if (!mounted) return;
      setState(() {
        _authSession = session;
        _machines = machines;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _busy = false;
        });
      }
    }
  }

  Future<bool> _showTofuDialog(String host, String fingerprint) async {
    final result = await showDialog<bool>(
      context: context,
      barrierDismissible: false,
      builder: (context) => AlertDialog(
        title: const Text('Security Warning'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('The server is using a self-signed certificate.'),
            const SizedBox(height: 12),
            const Text('SHA-256 Fingerprint:', style: TextStyle(fontWeight: FontWeight.bold)),
            SelectableText(
              fingerprint,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
            ),
            const SizedBox(height: 12),
            const Text('Do you want to trust this server?'),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('Trust & Connect'),
          ),
        ],
      ),
    );
    return result ?? false;
  }

  Future<void> _scanQr() async {
    final rawValue = await Navigator.of(context).push<String>(
      MaterialPageRoute(builder: (context) => const QrScannerScreen()),
    );

    if (rawValue == null) return;

    try {
      final deepLink = _service.parseShareDeepLink(rawValue);
      await _connectFromShareToken(
        serverUrl: deepLink.serverUrl,
        sessionToken: deepLink.sessionToken,
        refreshToken: deepLink.refreshToken,
      );
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
      });
    }
  }

  Future<void> _connectMachine(RemoteMachine machine) async {
    final profile = _selectedProfile;
    if (profile == null) return;

    setState(() {
      _busy = true;
      _error = null;
    });

    try {
      final session = await _service.ensureValidSession(_storage, profile);
      await _storage.savePublicAuthSession(profile.id, session);

      final host = Uri.parse(profile.apiBaseUrl).host;
      final trustedFingerprint = await _storage.getTrustedFingerprint(host);

      final share = await _service.createShareSession(
        profile,
        session.accessToken,
        machine.id,
        trustedFingerprint: trustedFingerprint,
      );

      if (!mounted) return;
      setState(() {
        _authSession = session;
      });

      await _connectFromShareToken(
        serverUrl: share.serverUrl,
        sessionToken: share.sessionToken,
        refreshToken: share.refreshToken,
      );
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _busy = false;
        });
      }
    }
  }

  Future<void> _connectFromShareToken({
    required String serverUrl,
    required String sessionToken,
    String? refreshToken,
  }) async {
    setState(() {
      _busy = true;
      _error = null;
    });

    try {
      final host = Uri.parse(serverUrl).host;
      String? trustedFingerprint = await _storage.getTrustedFingerprint(host);

      ShareConnectResponse details;
      try {
        details = await _service.connectShareSession(
          serverUrl,
          sessionToken,
          trustedFingerprint: trustedFingerprint,
        );
      } catch (e) {
        final errorMessage = e.toString().toLowerCase();
        
        if (errorMessage.contains('handshake') ||
            errorMessage.contains('certificate') ||
            errorMessage.contains('connection failed')) {
          
          final probedFingerprint = await _service.probeFingerprint(serverUrl);
          
          final confirmed = await _showTofuDialog(host, probedFingerprint);
          if (!confirmed) return;

          await _storage.saveTrustedFingerprint(host, probedFingerprint);
          trustedFingerprint = probedFingerprint;
          
          return _connectFromShareToken(
            serverUrl: serverUrl,
            sessionToken: sessionToken,
            refreshToken: refreshToken,
          );
        }
        rethrow;
      }

      if (details.serverTlsFingerprint != null &&
          details.serverTlsFingerprint != trustedFingerprint) {
        final confirmed = await _showTofuDialog(
          host,
          details.serverTlsFingerprint!,
        );
        if (!confirmed) return;

        await _storage.saveTrustedFingerprint(host, details.serverTlsFingerprint!);
        trustedFingerprint = details.serverTlsFingerprint;
      }

      final client = TerminalClient();
      final status = await client.connectToShareSession(
        details.relayUrl,
        sessionToken,
        serverUrl: serverUrl,
        refreshToken: refreshToken,
        expiresAt: details.expiresAt,
        expectedFingerprint: trustedFingerprint,
      );

      if (status != ConnectionStatus.connected) {
        if (status == ConnectionStatus.fingerprintMismatch) {
          throw Exception('Security Alert: Server certificate has changed!');
        }
        throw Exception(
          client.errorMessage ?? 'Failed to connect to the share session.',
        );
      }

      await widget.onConnected(client);
      await _refreshMachines();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _busy = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final selectedProfile = _selectedProfile;

    return Scaffold(
      appBar: AppBar(
        title: const Text('TermViewer Remote Access'),
        actions: [
          IconButton(
            onPressed: _busy ? null : _scanQr,
            icon: const Icon(Icons.qr_code_scanner),
            tooltip: 'Scan Share QR',
          ),
          IconButton(
            onPressed: _busy ? null : () => _openProfileEditor(),
            icon: const Icon(Icons.add_link),
            tooltip: 'Add Server Profile',
          ),
          IconButton(
            onPressed: _busy && !_loading ? null : _loadProfiles,
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                if (_error != null)
                  Card(
                    color: Colors.red.shade900.withValues(alpha: 0.4),
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Text(
                        _error!,
                        style: const TextStyle(color: Colors.white),
                      ),
                    ),
                  ),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text(
                          'Scan a Share QR',
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        const SizedBox(height: 8),
                        const Text(
                          'Use the in-app QR scanner to open a short-lived share session immediately, even without signing in first.',
                        ),
                        const SizedBox(height: 12),
                        FilledButton.icon(
                          onPressed: _busy ? null : _scanQr,
                          icon: const Icon(Icons.qr_code_scanner),
                          label: const Text('Scan Share QR'),
                        ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            const Expanded(
                              child: Text(
                                'Public Server Profiles',
                                style: TextStyle(
                                  fontSize: 18,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                            ),
                            if (selectedProfile != null)
                              IconButton(
                                onPressed: _busy
                                    ? null
                                    : () => _openProfileEditor(
                                        initialProfile: selectedProfile,
                                      ),
                                icon: const Icon(Icons.edit_outlined),
                                tooltip: 'Edit selected profile',
                              ),
                            if (selectedProfile != null)
                              IconButton(
                                onPressed: _busy
                                    ? null
                                    : _deleteSelectedProfile,
                                icon: const Icon(Icons.delete_outline),
                                tooltip: 'Delete selected profile',
                              ),
                          ],
                        ),
                        const SizedBox(height: 8),
                        if (_profiles.isEmpty)
                          const Text(
                            'No server profiles yet. Add your Keycloak/OIDC server to enable authenticated remote access.',
                          )
                        else ...[
                          DropdownButtonFormField<String>(
                            initialValue: _selectedProfileId,
                            decoration: const InputDecoration(
                              labelText: 'Selected server',
                              border: OutlineInputBorder(),
                            ),
                            items: _profiles
                                .map(
                                  (profile) => DropdownMenuItem<String>(
                                    value: profile.id,
                                    child: Text(profile.name),
                                  ),
                                )
                                .toList(),
                            onChanged: _busy
                                ? null
                                : (value) async {
                                    if (value == null) return;
                                    final session = await _storage
                                        .getPublicAuthSession(value);
                                    if (!mounted) return;
                                    setState(() {
                                      _selectedProfileId = value;
                                      _authSession = session;
                                      _machines = <RemoteMachine>[];
                                    });
                                    if (session != null) {
                                      await _refreshMachines();
                                    }
                                  },
                          ),
                          if (selectedProfile != null) ...[
                            const SizedBox(height: 16),
                            Text('API: ${selectedProfile.apiBaseUrl}'),
                            Text('OIDC issuer: ${selectedProfile.oidcIssuer}'),
                            Text('Client ID: ${selectedProfile.clientId}'),
                            const SizedBox(height: 12),
                            Wrap(
                              spacing: 12,
                              runSpacing: 12,
                              children: [
                                FilledButton.icon(
                                  onPressed: _busy
                                      ? null
                                      : () => _signIn(
                                          force: _authSession == null,
                                        ),
                                  icon: const Icon(Icons.lock_open),
                                  label: Text(
                                    _authSession == null
                                        ? 'Sign in with OIDC'
                                        : 'Refresh login',
                                  ),
                                ),
                                if (_authSession != null)
                                  OutlinedButton.icon(
                                    onPressed: _busy ? null : _signOut,
                                    icon: const Icon(Icons.logout),
                                    label: const Text('Sign out'),
                                  ),
                                if (_authSession != null)
                                  OutlinedButton.icon(
                                    onPressed: _busy ? null : _refreshMachines,
                                    icon: const Icon(Icons.computer),
                                    label: const Text('Load machines'),
                                  ),
                              ],
                            ),
                          ],
                        ],
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                if (_authSession != null && selectedProfile != null)
                  Card(
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text(
                            'Your Machines',
                            style: TextStyle(
                              fontSize: 18,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                          const SizedBox(height: 12),
                          if (_machines.isEmpty)
                            const Text(
                              'No machines loaded yet, or your account does not have any registered devices.',
                            )
                          else
                            ..._machines.map(
                              (machine) => Card(
                                margin: const EdgeInsets.only(bottom: 12),
                                child: ListTile(
                                  leading: const Icon(Icons.computer),
                                  title: Text(machine.name),
                                  subtitle: Text(
                                    'Status: ${machine.status.toUpperCase()}\n'
                                    'Last seen: ${machine.lastSeenAt?.toLocal().toString() ?? "Never"}',
                                  ),
                                  trailing: FilledButton(
                                    onPressed: !_busy && machine.canConnect
                                        ? () => _connectMachine(machine)
                                        : null,
                                    child: Text(
                                      machine.status == 'waiting'
                                          ? 'Refresh & Connect'
                                          : 'Connect',
                                    ),
                                  ),
                                ),
                              ),
                            ),
                        ],
                      ),
                    ),
                  ),
              ],
            ),
    );
  }
}
