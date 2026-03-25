import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:xterm/xterm.dart';
import '../terminal_client.dart';
import '../storage_service.dart';
import '../terminal_themes.dart';
import '../widgets/file_browser.dart';
import '../widgets/key_button.dart';
import '../widgets/tab_button.dart';
import '../widgets/hud_line.dart';

class TerminalScreen extends StatefulWidget {
  final TerminalClient client;
  const TerminalScreen({super.key, required this.client});

  @override
  State<TerminalScreen> createState() => _TerminalScreenState();
}

class _TerminalScreenState extends State<TerminalScreen> {
  final FocusNode _focusNode = FocusNode();
  final TransformationController _transformationController = TransformationController();
  bool _showKeyBar = true;
  StreamSubscription? _statusSubscription;
  StreamSubscription? _downloadSubscription;
  Timer? _clipboardTimer;
  int _activeTab = 0;
  List<Macro> _macros = [];
  bool _showHUD = false;
  ThemeDefinition _selectedTheme = terminalThemes.first;

  @override
  void initState() {
    super.initState();
    _loadMacros();
    _loadTheme();
    _statusSubscription = widget.client.statusStream.listen((status) {
      if (status == ConnectionStatus.disconnected && mounted) {
        Navigator.pop(context);
      }
    });

    _downloadSubscription = widget.client.downloadStream.listen((message) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(message), duration: const Duration(seconds: 3)),
        );
      }
    });

    _clipboardTimer = Timer.periodic(const Duration(seconds: 2), (timer) async {
      final data = await Clipboard.getData(Clipboard.kTextPlain);
      if (data?.text != null && mounted) {
        widget.client.updateClipboard(data!.text!);
      }
    });

    WidgetsBinding.instance.addPostFrameCallback((_) {
      _focusNode.requestFocus();
    });
  }

  Future<void> _loadTheme() async {
    final name = await StorageService().getSelectedThemeName();
    final theme = terminalThemes.firstWhere((t) => t.name == name, orElse: () => terminalThemes.first);
    if (mounted) {
      setState(() {
        _selectedTheme = theme;
      });
    }
  }

  Future<void> _loadMacros() async {
    final macros = await StorageService().getMacros();
    if (mounted) {
      setState(() {
        _macros = macros;
      });
    }
  }

  @override
  void dispose() {
    _statusSubscription?.cancel();
    _downloadSubscription?.cancel();
    _clipboardTimer?.cancel();
    _focusNode.dispose();
    super.dispose();
  }

  void _sendKey(String key) {
    widget.client.terminal.onOutput!(key);
  }

  void _resetView() {
    setState(() {
      _transformationController.value = Matrix4.identity();
    });
  }

  void _showMacrosSheet() {
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.grey[900],
      builder: (context) {
        return Container(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('Macros', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18, color: Colors.white)),
              const SizedBox(height: 10),
              Flexible(
                child: ListView.builder(
                  shrinkWrap: true,
                  itemCount: _macros.length,
                  itemBuilder: (context, index) {
                    final m = _macros[index];
                    return ListTile(
                      title: Text(m.name, style: const TextStyle(color: Colors.white)),
                      subtitle: Text(m.command, style: const TextStyle(fontFamily: 'monospace', fontSize: 12, color: Colors.white70)),
                      onTap: () {
                        _sendKey('${m.command}\n');
                        Navigator.pop(context);
                      },
                    );
                  },
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  void _showThemeSheet() {
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.grey[900],
      builder: (context) {
        return Container(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('Terminal Themes', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18, color: Colors.white)),
              const SizedBox(height: 10),
              Flexible(
                child: ListView.builder(
                  shrinkWrap: true,
                  itemCount: terminalThemes.length,
                  itemBuilder: (context, index) {
                    final t = terminalThemes[index];
                    final isSelected = t.name == _selectedTheme.name;
                    return ListTile(
                      leading: Icon(Icons.lens, color: t.theme.background),
                      title: Text(t.name, style: TextStyle(color: isSelected ? Colors.blueAccent : Colors.white, fontWeight: isSelected ? FontWeight.bold : FontWeight.normal)),
                      trailing: isSelected ? const Icon(Icons.check, color: Colors.blueAccent) : null,
                      onTap: () async {
                        final navigator = Navigator.of(context);
                        setState(() {
                          _selectedTheme = t;
                        });
                        await StorageService().saveSelectedThemeName(t.name);
                        navigator.pop();
                      },
                    );
                  },
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        title: StreamBuilder<ConnectionStatus>(
          stream: widget.client.statusStream,
          initialData: widget.client.status,
          builder: (context, snapshot) {
            final status = snapshot.data?.name.toUpperCase() ?? 'UNKNOWN';
            return Text('TermViewer - $status', style: const TextStyle(fontSize: 14));
          },
        ),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(40),
          child: Row(
            children: [
              TabButton(
                label: 'TERMINAL', 
                isActive: _activeTab == 0, 
                onTap: () => setState(() => _activeTab = 0)
              ),
              TabButton(
                label: 'FILES', 
                isActive: _activeTab == 1, 
                onTap: () {
                  setState(() => _activeTab = 1);
                  widget.client.listFiles('~');
                }
              ),
            ],
          ),
        ),
        actions: [
          if (_activeTab == 0) ...[
            StreamBuilder<bool>(
              stream: widget.client.recordStream,
              initialData: widget.client.isRecording,
              builder: (context, snapshot) {
                final isRec = snapshot.data ?? false;
                return IconButton(
                  icon: Icon(isRec ? Icons.fiber_manual_record : Icons.fiber_manual_record_outlined, color: isRec ? Colors.red : null),
                  onPressed: () => widget.client.toggleRecording(!isRec),
                  tooltip: isRec ? 'Stop Recording' : 'Start Recording',
                );
              },
            ),
            IconButton(
              icon: const Icon(Icons.palette_outlined),
              onPressed: _showThemeSheet,
              tooltip: 'Terminal Theme',
            ),
            IconButton(
              icon: Icon(_showHUD ? Icons.analytics : Icons.analytics_outlined),
              onPressed: () => setState(() => _showHUD = !_showHUD),
              tooltip: 'Toggle System HUD',
            ),
            IconButton(
              icon: const Icon(Icons.fullscreen_exit),
              onPressed: _resetView,
              tooltip: 'Reset View',
            ),
            IconButton(
              icon: Icon(_showKeyBar ? Icons.keyboard : Icons.keyboard_hide),
              onPressed: () {
                setState(() {
                  _showKeyBar = !_showKeyBar;
                });
                _focusNode.requestFocus();
                SystemChannels.textInput.invokeMethod('TextInput.show');
              },
            ),
          ],
          IconButton(
            icon: const Icon(Icons.close),
            onPressed: () {
              widget.client.disconnect();
              Navigator.pop(context);
            },
          )
        ],
      ),
      body: SafeArea(
        child: Stack(
          children: [
            _activeTab == 0 ? _buildTerminal() : FileBrowser(client: widget.client),
            if (_showHUD && _activeTab == 0) _buildHUD(),
          ],
        ),
      ),
    );
  }

  Widget _buildHUD() {
    return Positioned(
      top: 10,
      right: 10,
      child: StreamBuilder<SystemStats>(
        stream: widget.client.statsStream,
        builder: (context, snapshot) {
          if (!snapshot.hasData) return const SizedBox.shrink();
          final s = snapshot.data!;
          return Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: Colors.black.withValues(alpha: 0.7),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.blueAccent.withValues(alpha: 0.5)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                HUDLine(label: 'CPU', value: '${s.cpuUsage.toStringAsFixed(1)}%'),
                HUDLine(label: 'RAM', value: '${s.ramUsedGb.toStringAsFixed(1)} / ${s.ramTotalGb.toStringAsFixed(1)} GB'),
                HUDLine(label: 'DISK', value: '${s.diskPercent.toStringAsFixed(1)}%'),
                HUDLine(label: 'UPTIME', value: '${(s.uptimeSeconds / 3600).toStringAsFixed(1)}h'),
              ],
            ),
          );
        },
      ),
    );
  }

  Widget _buildTerminal() {
    return Column(
      children: [
        Expanded(
          child: LayoutBuilder(
            builder: (context, constraints) {
              return InteractiveViewer(
                transformationController: _transformationController,
                minScale: 0.5,
                maxScale: 2.0,
                boundaryMargin: const EdgeInsets.all(20),
                constrained: false,
                child: Container(
                  padding: const EdgeInsets.all(8),
                  width: 1200,
                  height: constraints.maxHeight,
                  child: TerminalView(
                    widget.client.terminal,
                    controller: widget.client.controller,
                    focusNode: _focusNode,
                    autofocus: true,
                    backgroundOpacity: 0,
                    theme: _selectedTheme.theme,
                    textStyle: const TerminalStyle(
                      fontFamily: 'FiraCode',
                      fontSize: 12,
                    ),
                  ),
                ),
              );
            },
          ),
        ),
        if (_showKeyBar)
          Container(
            color: Colors.grey[900],
            padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Row(
                children: [
                  KeyButton(
                    label: 'COPY',
                    onTap: () async {
                      final selection = widget.client.controller.selection;
                      if (selection != null) {
                        final text = widget.client.terminal.buffer.getText(selection);
                        if (text.isNotEmpty) {
                          await Clipboard.setData(ClipboardData(text: text));
                          if (!mounted) return;
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(content: Text('Copied to clipboard'), duration: Duration(seconds: 1)),
                          );
                        }
                      }
                    },
                  ),
                  KeyButton(
                    label: 'PASTE',
                    onTap: () async {
                      final data = await Clipboard.getData(Clipboard.kTextPlain);
                      if (data?.text != null) {
                        _sendKey(data!.text!);
                      }
                    },
                  ),
                  const VerticalDivider(color: Colors.white24),
                  KeyButton(label: 'ESC', onTap: () => _sendKey('\x1b')),
                  KeyButton(label: 'TAB', onTap: () => _sendKey('\t')),
                  KeyButton(label: 'CTRL', onTap: () => _sendKey('\x03')),
                  KeyButton(label: 'UP', onTap: () => _sendKey('\x1b[A')),
                  KeyButton(label: 'DOWN', onTap: () => _sendKey('\x1b[B')),
                  KeyButton(label: 'LEFT', onTap: () => _sendKey('\x1b[D')),
                  KeyButton(label: 'RIGHT', onTap: () => _sendKey('\x1b[C')),
                  KeyButton(label: ':', onTap: () => _sendKey(':')),
                  KeyButton(label: '/', onTap: () => _sendKey('/')),
                  const VerticalDivider(color: Colors.white24),
                  KeyButton(
                    label: 'MACROS',
                    onTap: () {
                      _showMacrosSheet();
                    },
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}
