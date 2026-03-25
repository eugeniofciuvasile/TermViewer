import 'package:flutter/material.dart';
import 'package:path/path.dart' as path;
import 'package:file_picker/file_picker.dart';
import '../terminal_client.dart';

class FileBrowser extends StatefulWidget {
  final TerminalClient client;
  const FileBrowser({super.key, required this.client});

  @override
  State<FileBrowser> createState() => _FileBrowserState();
}

class _FileBrowserState extends State<FileBrowser> {
  String _currentPath = '';

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<Map<String, dynamic>>(
      stream: widget.client.fileListStream,
      builder: (context, snapshot) {
        if (!snapshot.hasData) {
          return const Center(child: CircularProgressIndicator());
        }

        final data = snapshot.data!;
        _currentPath = data['path'];
        final List<dynamic> filesJson = data['files'] ?? [];
        final List<RemoteFile> files = filesJson.map((f) => RemoteFile.fromJson(f)).toList();

        files.sort((a, b) {
          if (a.isDir && !b.isDir) return -1;
          if (!a.isDir && b.isDir) return 1;
          return a.name.toLowerCase().compareTo(b.name.toLowerCase());
        });

        return Column(
          children: [
            Container(
              padding: const EdgeInsets.all(8),
              color: Colors.blueGrey[900],
              child: Row(
                children: [
                  IconButton(
                    icon: const Icon(Icons.arrow_upward, size: 20),
                    onPressed: () {
                      final parent = path.dirname(_currentPath);
                      widget.client.listFiles(parent);
                    },
                  ),
                  Expanded(
                    child: Text(
                      _currentPath,
                      style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.refresh, size: 20),
                    onPressed: () => widget.client.listFiles(_currentPath),
                  ),
                  IconButton(
                    icon: const Icon(Icons.upload_file, size: 20, color: Colors.blueAccent),
                    onPressed: () async {
                      FilePickerResult? result = await FilePicker.platform.pickFiles(allowMultiple: true);
                      if (result != null) {
                        for (var file in result.files) {
                          if (file.path != null) {
                            widget.client.uploadFile(file.path!, _currentPath);
                          }
                        }
                      }
                    },
                    tooltip: 'Upload to this folder',
                  ),
                ],
              ),
            ),
            Expanded(
              child: ListView.builder(
                itemCount: files.length,
                itemBuilder: (context, index) {
                  final file = files[index];
                  return ListTile(
                    dense: true,
                    leading: Icon(
                      file.isDir ? Icons.folder : Icons.insert_drive_file,
                      color: file.isDir ? Colors.amber : Colors.blueGrey[300],
                    ),
                    title: Text(file.name, style: const TextStyle(fontSize: 13)),
                    subtitle: Text(
                      file.isDir ? '' : '${(file.size / 1024).toStringAsFixed(1)} KB',
                      style: const TextStyle(fontSize: 11),
                    ),
                    onTap: () {
                      if (file.isDir) {
                        widget.client.listFiles(path.join(_currentPath, file.name));
                      } else {
                        ScaffoldMessenger.of(context).showSnackBar(
                          SnackBar(content: Text('Download started for ${file.name}')),
                        );
                        widget.client.downloadFile(path.join(_currentPath, file.name));
                      }
                    },
                  );
                },
              ),
            ),
          ],
        );
      },
    );
  }
}
