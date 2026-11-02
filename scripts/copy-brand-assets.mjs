import { copyFile } from 'node:fs/promises';
import { resolve } from 'node:path';
const root = resolve(import.meta.dirname, '..');
const output = resolve(root, 'build/vite-web');
for (const name of ['nexora-logo-v21.png', 'login-banner-v21.png', 'nexora-poster-v21.png']) {
  await copyFile(resolve(root, 'web', name), resolve(output, name));
}
