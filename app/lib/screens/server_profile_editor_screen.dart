import 'package:flutter/material.dart';
import '../storage_service.dart';
import '../public_access_service.dart';

class ServerProfileEditorScreen extends StatefulWidget {
  const ServerProfileEditorScreen({super.key, this.initialProfile});

  final ServerProfile? initialProfile;

  @override
  State<ServerProfileEditorScreen> createState() =>
      _ServerProfileEditorScreenState();
}

class _ServerProfileEditorScreenState extends State<ServerProfileEditorScreen> {
  late final TextEditingController _nameController;
  late final TextEditingController _apiBaseUrlController;
  late final TextEditingController _oidcIssuerController;
  late final TextEditingController _clientIdController;
  late final TextEditingController _redirectUrlController;
  late final TextEditingController _scopesController;
  late final TextEditingController _tlsFingerprintController;

  @override
  void initState() {
    super.initState();
    final initial = widget.initialProfile;
    _nameController = TextEditingController(text: initial?.name ?? '');
    _apiBaseUrlController = TextEditingController(
      text: initial?.apiBaseUrl ?? 'https://termviewer.local',
    );
    _oidcIssuerController = TextEditingController(
      text: initial?.oidcIssuer ?? 'https://sso.termviewer.local/realms/termviewer',
    );
    _clientIdController = TextEditingController(
      text: initial?.clientId ?? 'termviewer-app',
    );
    _redirectUrlController = TextEditingController(
      text: initial?.redirectUrl ?? defaultRedirectUrl,
    );
    _scopesController = TextEditingController(
      text:
          (initial?.scopes.isNotEmpty == true
                  ? initial!.scopes
                  : defaultOidcScopes)
              .join(' '),
    );
    _tlsFingerprintController = TextEditingController(
      text: initial?.tlsFingerprint ?? '',
    );
  }

  @override
  void dispose() {
    _nameController.dispose();
    _apiBaseUrlController.dispose();
    _oidcIssuerController.dispose();
    _clientIdController.dispose();
    _redirectUrlController.dispose();
    _scopesController.dispose();
    _tlsFingerprintController.dispose();
    super.dispose();
  }

  void _submit() {
    if (_nameController.text.trim().isEmpty ||
        _apiBaseUrlController.text.trim().isEmpty ||
        _oidcIssuerController.text.trim().isEmpty ||
        _clientIdController.text.trim().isEmpty ||
        _redirectUrlController.text.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Please fill in all required profile fields.'),
        ),
      );
      return;
    }

    final profile = ServerProfile(
      id:
          widget.initialProfile?.id ??
          DateTime.now().millisecondsSinceEpoch.toString(),
      name: _nameController.text.trim(),
      apiBaseUrl: _apiBaseUrlController.text.trim(),
      oidcIssuer: _oidcIssuerController.text.trim(),
      clientId: _clientIdController.text.trim(),
      redirectUrl: _redirectUrlController.text.trim(),
      scopes: _scopesController.text
          .split(RegExp(r'\s+'))
          .map((scope) => scope.trim())
          .where((scope) => scope.isNotEmpty)
          .toList(),
      tlsFingerprint: _tlsFingerprintController.text.trim().isEmpty
          ? null
          : _tlsFingerprintController.text.trim(),
    );

    Navigator.of(context).pop(profile);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(
          widget.initialProfile == null
              ? 'Add Server Profile'
              : 'Edit Server Profile',
        ),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(
            controller: _nameController,
            decoration: const InputDecoration(
              labelText: 'Profile name',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _apiBaseUrlController,
            decoration: const InputDecoration(
              labelText: 'API base URL',
              helperText: 'Example: https://api.termviewer.example',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _oidcIssuerController,
            decoration: const InputDecoration(
              labelText: 'OIDC issuer URL',
              helperText: 'Example: https://auth.termviewer.example/realms/termviewer',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _tlsFingerprintController,
            decoration: const InputDecoration(
              labelText: 'TLS Fingerprint (Optional)',
              helperText: 'SHA-256 hash if using self-signed certificates.',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _clientIdController,
            decoration: const InputDecoration(
              labelText: 'OIDC client ID',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _redirectUrlController,
            decoration: const InputDecoration(
              labelText: 'Redirect URI',
              helperText:
                  'Must stay registered in the app and in your Keycloak client.',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _scopesController,
            decoration: const InputDecoration(
              labelText: 'Scopes',
              helperText:
                  'Space-separated scopes. Default: openid profile email offline_access',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 24),
          FilledButton(
            onPressed: _submit,
            child: Text(
              widget.initialProfile == null ? 'Save Profile' : 'Update Profile',
            ),
          ),
        ],
      ),
    );
  }
}
