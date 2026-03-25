// This is a basic Flutter widget test.
//
// To perform an interaction with a widget in your test, use the WidgetTester
// utility in the flutter_test package. For example, you can send tap and scroll
// gestures. You can also use WidgetTester to find child widgets in the widget
// tree, read text, and verify that the values of widget properties are correct.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:termviewer/main.dart';

void main() {
  testWidgets('home screen shows LAN and remote entry points', (WidgetTester tester) async {
    await tester.pumpWidget(const TermViewerApp());

    expect(find.text('TermViewer: Connect to PC'), findsOneWidget);
    expect(find.text('Connect Manually'), findsOneWidget);
    expect(find.text('Remote Access (OIDC / QR)'), findsOneWidget);
    expect(find.byIcon(Icons.public), findsWidgets);
  });
}
