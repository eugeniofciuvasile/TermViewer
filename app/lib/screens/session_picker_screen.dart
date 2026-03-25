import 'package:flutter/material.dart';
import '../terminal_client.dart';
import 'terminal_screen.dart';

class SessionPickerScreen extends StatelessWidget {
  final TerminalClient client;
  const SessionPickerScreen({super.key, required this.client});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Choose Terminal Session'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () {
            client.disconnect();
            Navigator.pop(context);
          },
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () => client.listSessions(),
            tooltip: 'Refresh Sessions',
          ),
        ],
      ),
      body: StreamBuilder<List<TerminalSession>>(
        stream: client.sessionsStream,
        initialData: client.currentSessions,
        builder: (context, snapshot) {
          if (!snapshot.hasData || (snapshot.data!.isEmpty && client.status == ConnectionStatus.connecting)) {
            return const Center(child: CircularProgressIndicator());
          }

          final sessions = snapshot.data!;
          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              ...sessions.map((session) => Card(
                clipBehavior: Clip.antiAlias,
                child: InkWell(
                  onTap: () async {
                    client.initTerminal(sessionId: session.id);
                    await Navigator.of(context).push(
                      MaterialPageRoute(
                        builder: (context) => TerminalScreen(client: client),
                      ),
                    );
                    client.listSessions();
                  },
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      ListTile(
                        leading: Icon(
                          session.type == 'tmux' ? Icons.slideshow : Icons.input,
                          color: session.type == 'tmux' ? Colors.greenAccent : Colors.orangeAccent,
                        ),
                        title: Text(session.name),
                        subtitle: Text(
                          session.type == 'tmux' 
                            ? 'MODE: SHARED' 
                            : 'MODE: HIJACK',
                          style: TextStyle(
                            color: session.type == 'tmux' ? Colors.green[200] : Colors.orange[200],
                            fontSize: 10,
                          ),
                        ),
                        trailing: const Icon(Icons.chevron_right),
                      ),
                      Container(
                        padding: const EdgeInsets.all(8),
                        color: Colors.black87,
                        width: double.infinity,
                        height: 100,
                        child: Text(
                          session.context,
                          style: const TextStyle(
                            color: Colors.greenAccent,
                            fontFamily: 'monospace',
                            fontSize: 10,
                          ),
                          maxLines: 6,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    ],
                  ),
                ),
              )),
              const SizedBox(height: 16),
              ElevatedButton.icon(
                onPressed: () async {
                  client.initTerminal(command: 'bash');
                  await Navigator.of(context).push(
                    MaterialPageRoute(
                      builder: (context) => TerminalScreen(client: client),
                    ),
                  );
                  client.listSessions();
                },
                icon: const Icon(Icons.add),
                label: const Text('Start New Session'),
                style: ElevatedButton.styleFrom(
                  minimumSize: const Size.fromHeight(50),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}
