import 'dart:async';
import 'package:flutter/material.dart';
import 'package:nsd/nsd.dart';
import '../terminal_client.dart';
import '../storage_service.dart';
import 'session_picker_screen.dart';
import 'remote_access_screen.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  final List<Service> _services = [];
  List<SavedConnection> _savedConnections = [];
  Discovery? _discovery;

  final TextEditingController _hostController = TextEditingController();
  final TextEditingController _portController = TextEditingController(text: '24242');
  final TextEditingController _passwordController = TextEditingController();

  final TerminalClient _client = TerminalClient();
  final StorageService _storage = StorageService();

  @override
  void initState() {
    super.initState();
    _startDiscovery();
    _loadSavedConnections();
  }

  Future<void> _loadSavedConnections() async {
    final conns = await _storage.getConnections();
    if (mounted) {
      setState(() {
        _savedConnections = conns;
      });
    }
  }

  Future<void> _startDiscovery() async {
    try {
      if (_discovery != null) {
        await stopDiscovery(_discovery!);
      }
      _discovery = await startDiscovery('_termviewer._tcp');
      _discovery!.addListener(() {
        if (mounted) {
          setState(() {
            _services.clear();
            _services.addAll(_discovery!.services);
          });
        }
      });
    } catch (e) {
      debugPrint('mDNS discovery not supported on this platform: $e');
    }
  }

  Future<String?> _showPasswordDialog() async {
    String? password;
    return showDialog<String>(
      context: context,
      barrierDismissible: false,
      builder: (BuildContext context) {
        return AlertDialog(
          title: const Text('Authentication Required'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Text('The agent requires a password to connect.'),
              TextField(
                obscureText: true,
                autofocus: true,
                decoration: const InputDecoration(labelText: 'Password'),
                onChanged: (value) => password = value,
                onSubmitted: (value) => Navigator.of(context).pop(value),
              ),
            ],
          ),
          actions: <Widget>[
            TextButton(
              child: const Text('Cancel'),
              onPressed: () => Navigator.of(context).pop(),
            ),
            TextButton(
              child: const Text('Connect'),
              onPressed: () => Navigator.of(context).pop(password),
            ),
          ],
        );
      },
    );
  }

  Future<bool> _showFingerprintMismatchDialog() async {
    return await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('SECURITY WARNING'),
        content: const Text('The host\'s security certificate has changed! This could be a Man-in-the-Middle attack. Do not connect unless you know the host was re-installed.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('Cancel')),
          TextButton(onPressed: () => Navigator.pop(context, true), child: const Text('Connect Anyway', style: TextStyle(color: Colors.red))),
        ],
      ),
    ) ?? false;
  }

  Future<bool> _showNewFingerprintConfirmDialog(String fingerprint) async {
    return await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Trust New Host?'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('First time connecting to this host. Verify the fingerprint matches the one shown on your PC:'),
            const SizedBox(height: 10),
            Container(
              padding: const EdgeInsets.all(8),
              color: Colors.black54,
              child: Text(fingerprint, style: const TextStyle(fontFamily: 'monospace', fontSize: 12, color: Colors.amber)),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('Cancel')),
          TextButton(onPressed: () => Navigator.pop(context, true), child: const Text('Trust & Connect')),
        ],
      ),
    ) ?? false;
  }

  void _connect(String host, int port, {String? storedPassword, String? expectedFingerprint}) async {
    String pwdToTry = storedPassword ?? _passwordController.text;
    
    if (pwdToTry.isEmpty) {
       final savedPwd = await _storage.getPasswordFor(host, port);
       if (savedPwd != null) pwdToTry = savedPwd;
    }

    var status = await _client.connect(host, port, pwdToTry, expectedFingerprint: expectedFingerprint);
    
    if (status == ConnectionStatus.fingerprintMismatch) {
      final proceed = await _showFingerprintMismatchDialog();
      if (!proceed) return;
      status = await _client.connect(host, port, pwdToTry);
    }

    if (status == ConnectionStatus.error && _client.errorMessage == 'Invalid password') {
      final newPassword = await _showPasswordDialog();
      if (newPassword != null) {
        pwdToTry = newPassword;
        status = await _client.connect(host, port, pwdToTry, expectedFingerprint: expectedFingerprint);
      }
    }

    if (!mounted) return;
    
    if (status == ConnectionStatus.connected) {
      final currentFingerprint = _client.serverFingerprint;
      
      if (expectedFingerprint == null && currentFingerprint != null) {
        final trust = await _showNewFingerprintConfirmDialog(currentFingerprint);
        if (!trust) {
          _client.disconnect();
          return;
        }
      }

      await _storage.saveConnection(SavedConnection(
        host: host, 
        port: port, 
        password: pwdToTry,
        fingerprint: currentFingerprint,
      ));
      _loadSavedConnections();
      
      if (!mounted) return;
      Navigator.of(context).push(
        MaterialPageRoute(
          builder: (context) => SessionPickerScreen(client: _client),
        ),
      );
    } else if (status == ConnectionStatus.error) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Connection failed: ${_client.errorMessage}')),
      );
    }
  }

  Future<void> _openRemoteAccess() async {
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (context) => RemoteAccessScreen(
          onConnected: (client) async {
            await Navigator.of(context).push(
              MaterialPageRoute(
                builder: (context) => SessionPickerScreen(client: client),
              ),
            );
          },
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('TermViewer: Connect to PC'),
        actions: [
          IconButton(
            icon: const Icon(Icons.public),
            onPressed: _openRemoteAccess,
            tooltip: 'Remote Access',
          ),
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () {
              _startDiscovery();
              _loadSavedConnections();
            },
            tooltip: 'Refresh Agents',
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (_savedConnections.isNotEmpty) ...[
              const Text('Saved Connections:', style: TextStyle(fontWeight: FontWeight.bold)),
              const SizedBox(height: 8),
              ListView.builder(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: _savedConnections.length,
                itemBuilder: (context, index) {
                  final sc = _savedConnections[index];
                  return Card(
                    child: ListTile(
                      leading: const Icon(Icons.bookmark),
                      title: Text('${sc.host}:${sc.port}'),
                      trailing: IconButton(
                        icon: const Icon(Icons.delete, color: Colors.redAccent),
                        onPressed: () async {
                          await _storage.deleteConnection(sc.host, sc.port);
                          _loadSavedConnections();
                        },
                      ),
                      onTap: () {
                        _hostController.text = sc.host;
                        _portController.text = sc.port.toString();
                        _connect(sc.host, sc.port, storedPassword: sc.password, expectedFingerprint: sc.fingerprint);
                      },
                    ),
                  );
                },
              ),
              const Divider(height: 30),
            ],
            const Text('Manual Connect:', style: TextStyle(fontWeight: FontWeight.bold)),
            TextField(
              controller: _hostController,
              decoration: const InputDecoration(labelText: 'Host IP'),
            ),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _portController,
                    decoration: const InputDecoration(labelText: 'Port'),
                    keyboardType: TextInputType.number,
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: TextField(
                    controller: _passwordController,
                    decoration: const InputDecoration(labelText: 'Password'),
                    obscureText: true,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            ElevatedButton(
              onPressed: () => _connect(_hostController.text, int.parse(_portController.text)),
              child: const Text('Connect Manually'),
            ),
            const SizedBox(height: 10),
            OutlinedButton.icon(
              onPressed: _openRemoteAccess,
              icon: const Icon(Icons.public),
              label: const Text('Remote Access (OIDC / QR)'),
            ),
            const Divider(height: 30),
            const Text('Discovered Agents on LAN:', style: TextStyle(fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
            Expanded(
              child: _services.isEmpty 
                ? const Center(child: Text('Searching for TermViewer Agents...'))
                : ListView.builder(
                    itemCount: _services.length,
                    itemBuilder: (context, index) {
                      final service = _services[index];
                      return Card(
                        child: ListTile(
                          leading: const Icon(Icons.computer),
                          title: Text(service.name ?? 'Unknown Agent'),
                          subtitle: Text('${service.host}:${service.port}'),
                          onTap: () async {
                            _hostController.text = service.host!;
                            _portController.text = service.port!.toString();
                            
                            final saved = _savedConnections.where((c) => c.host == service.host && c.port == service.port);
                            if (saved.isNotEmpty) {
                              _connect(service.host!, service.port!, storedPassword: saved.first.password, expectedFingerprint: saved.first.fingerprint);
                            } else {
                              _connect(service.host!, service.port!);
                            }
                          },
                        ),
                      );
                    },
                  ),
            ),
          ],
        ),
      ),
    );
  }

  @override
  void dispose() {
    if (_discovery != null) {
      stopDiscovery(_discovery!);
    }
    _hostController.dispose();
    _portController.dispose();
    _passwordController.dispose();
    _client.disconnect();
    super.dispose();
  }
}
