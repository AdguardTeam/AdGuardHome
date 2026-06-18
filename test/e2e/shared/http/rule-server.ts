import http from 'node:http';

export interface RuleServer {
  baseUrl: string;
  /** URL reachable from inside a Docker container (host.docker.internal). */
  containerBaseUrl: string;
  close: () => Promise<void>;
}

export function resolveRuleServerBody(files: Record<string, string>, requestPath: string): string | undefined {
  return Object.prototype.hasOwnProperty.call(files, requestPath)
    ? files[requestPath]
    : undefined;
}

export async function startRuleServer(files: Record<string, string>): Promise<RuleServer> {
  const server = http.createServer((request, response) => {
    const requestPath = request.url ?? '/';
    const body = resolveRuleServerBody(files, requestPath);
    if (body === undefined) {
      response.statusCode = 404;
      response.end('not found');
      return;
    }

    response.statusCode = 200;
    response.setHeader('Content-Type', 'text/plain; charset=utf-8');
    response.end(body);
  });

  return new Promise<RuleServer>((resolve, reject) => {
    const onError = (error: Error) => {
      server.close(() => reject(error));
    };

    server.once('error', onError);
    // Bind all interfaces so AdGuard Home running inside a container can reach
    // the rule server through host.docker.internal.
    server.listen(0, '0.0.0.0', () => {
      const address = server.address();
      if (!address || typeof address === 'string') {
        reject(new Error('Failed to bind rule server'));
        return;
      }

      server.removeListener('error', onError);
      resolve({
        baseUrl: `http://127.0.0.1:${address.port}`,
        containerBaseUrl: `http://host.docker.internal:${address.port}`,
        close: () => new Promise((closeResolve, closeReject) => {
          server.close((error) => {
            if (error) {
              closeReject(error);
              return;
            }

            closeResolve();
          });
        }),
      });
    });
  });
}
