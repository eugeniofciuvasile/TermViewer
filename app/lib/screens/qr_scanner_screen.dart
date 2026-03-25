import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

class QrScannerScreen extends StatefulWidget {
  const QrScannerScreen({super.key});

  @override
  State<QrScannerScreen> createState() => _QrScannerScreenState();
}

class _QrScannerScreenState extends State<QrScannerScreen> {
  final MobileScannerController _controller = MobileScannerController(
    autoZoom: true,
    detectionSpeed: DetectionSpeed.noDuplicates,
    formats: const [BarcodeFormat.qrCode],
  );
  bool _handled = false;

  Future<void> _handleRawValue(String rawValue) async {
    if (_handled || !mounted) {
      return;
    }

    _handled = true;
    await HapticFeedback.selectionClick();
    await _controller.stop();

    if (!mounted) {
      return;
    }

    Navigator.of(context).pop(rawValue);
  }

  Future<void> _pasteFromClipboard() async {
    final clipboardData = await Clipboard.getData(Clipboard.kTextPlain);
    final text = clipboardData?.text?.trim();
    if (text == null || text.isEmpty || !mounted) {
      return;
    }

    await _handleRawValue(text);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<MobileScannerState>(
      valueListenable: _controller,
      builder: (context, scannerState, _) {
        final torchState = scannerState.torchState;

        return Scaffold(
          appBar: AppBar(
            title: const Text('Scan Share QR'),
            actions: [
              IconButton(
                onPressed: torchState == TorchState.unavailable
                    ? null
                    : _controller.toggleTorch,
                icon: Icon(
                  torchState == TorchState.on
                      ? Icons.flash_on
                      : Icons.flash_off,
                ),
                tooltip: 'Toggle flash',
              ),
              IconButton(
                onPressed: _pasteFromClipboard,
                icon: const Icon(Icons.content_paste),
                tooltip: 'Paste QR payload',
              ),
            ],
          ),
          body: LayoutBuilder(
            builder: (context, constraints) {
              final layoutSize = constraints.biggest;
              final scanSize = math.min(layoutSize.width * 0.72, 320.0);
              final scanWindow = Rect.fromCenter(
                center: layoutSize.center(Offset.zero),
                width: scanSize,
                height: scanSize,
              );

              return Stack(
                fit: StackFit.expand,
                children: [
                  MobileScanner(
                    controller: _controller,
                    scanWindow: scanWindow,
                    tapToFocus: true,
                    overlayBuilder: (context, constraints) => ScanWindowOverlay(
                      controller: _controller,
                      scanWindow: scanWindow,
                      borderColor: Theme.of(context).colorScheme.primary,
                      borderRadius: BorderRadius.circular(28),
                      borderWidth: 4,
                      color: Colors.black.withValues(alpha: 0.56),
                    ),
                    onDetect: (capture) {
                      if (_handled) {
                        return;
                      }

                      for (final barcode in capture.barcodes) {
                        final rawValue = barcode.rawValue?.trim();
                        if (rawValue != null && rawValue.isNotEmpty) {
                          _handleRawValue(rawValue);
                          return;
                        }
                      }
                    },
                  ),
                  SafeArea(
                    child: Padding(
                      padding: const EdgeInsets.all(20),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          Align(
                            alignment: Alignment.topCenter,
                            child: DecoratedBox(
                              decoration: BoxDecoration(
                                color: Colors.black.withValues(alpha: 0.45),
                                borderRadius: BorderRadius.circular(18),
                              ),
                              child: const Padding(
                                padding: EdgeInsets.symmetric(
                                  horizontal: 14,
                                  vertical: 10,
                                ),
                                child: Text(
                                  'Center the QR inside the frame. Tap the preview to refocus if needed.',
                                  textAlign: TextAlign.center,
                                  style: TextStyle(color: Colors.white),
                                ),
                              ),
                            ),
                          ),
                          const Spacer(),
                          Card(
                            child: Padding(
                              padding: const EdgeInsets.all(16),
                              child: Column(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Text(
                                    torchState == TorchState.unavailable
                                        ? 'Automatic zoom is enabled for easier QR recognition.'
                                        : 'Automatic zoom is enabled. Use flash if the screen or room is dim.',
                                    textAlign: TextAlign.center,
                                  ),
                                  const SizedBox(height: 10),
                                  const Text(
                                    'You can also paste the QR payload from the clipboard.',
                                    textAlign: TextAlign.center,
                                  ),
                                ],
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ],
              );
            },
          ),
        );
      },
    );
  }
}
