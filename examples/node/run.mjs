#!/usr/bin/env bun

// Example: running hackpad WASM in Node.js with overlayNodefs + overlayMemFS
// + spawning a child Go WASM process.
//
// Build:
//   PATH=cache/go/bin:cache/go/misc/wasm:$PATH GOOS=js GOARCH=wasm \
//     go build -o examples/node/main.wasm ./cmd/init/
//   PATH=cache/go/bin:cache/go/misc/wasm:$PATH GOOS=js GOARCH=wasm \
//     go build -o examples/node/info.wasm ./examples/node/info/
//   cp server/public/wasm/wasm_exec.js examples/node/
//
// Run:
//   node examples/node/run.mjs

import { readFileSync, writeFileSync, mkdirSync, statSync, lstatSync, readdirSync,
         unlinkSync, rmdirSync, rmSync, renameSync, chmodSync, utimesSync,
         readlinkSync, accessSync, symlinkSync } from 'fs';
import { createRequire } from 'module';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const __dirname = dirname(fileURLToPath(import.meta.url));

// ========================= SETUP =========================

// 1. Inject __nodefs bridge (required by overlayNodefs)
globalThis.__nodefs = (() => {
  const statToObj = s => ({
    dev: s.dev, ino: s.ino, mode: s.mode, nlink: s.nlink,
    uid: s.uid, gid: s.gid, rdev: s.rdev, size: s.size,
    blksize: s.blksize, blocks: s.blocks,
    isDirectory: s.isDirectory(), isFile: s.isFile(),
    isSymbolicLink: s.isSymbolicLink(),
    atimeMs: s.atimeMs, mtimeMs: s.mtimeMs,
  });
  return {
    readFileSync(p)     { return { data: new Uint8Array(readFileSync(p)).buffer, err: null }; },
    writeFileSync(p, b) { writeFileSync(p, Buffer.from(b)); return { err: null }; },
    statSync(p)         { return { stat: statToObj(statSync(p)), err: null }; },
    lstatSync(p)        { return { stat: statToObj(lstatSync(p)), err: null }; },
    mkdirSync(p, m)     { mkdirSync(p, { mode: m }); return { err: null }; },
    mkdirAllSync(p, m)  { mkdirSync(p, { recursive: true, mode: m }); return { err: null }; },
    readdirSync(p)      { const e = readdirSync(p, { withFileTypes: true }); return { entries: e.map(d => ({ name: d.name, isDirectory: d.isDirectory(), isSymbolicLink: d.isSymbolicLink() })), err: null }; },
    unlinkSync(p)       { unlinkSync(p); return { err: null }; },
    rmdirSync(p)        { rmdirSync(p); return { err: null }; },
    rmSync(p, r)        { rmSync(p, { recursive: r, force: true }); return { err: null }; },
    renameSync(o, n)    { renameSync(o, n); return { err: null }; },
    chmodSync(p, m)     { chmodSync(p, m); return { err: null }; },
    utimesSync(p, a, m) { utimesSync(p, a, m); return { err: null }; },
    symlinkSync(t, p)   { symlinkSync(t, p); return { err: null }; },
    readlinkSync(p)     { return { link: readlinkSync(p), err: null }; },
    accessSync(p, m)    { accessSync(p, m); return { err: null }; },
  };
})();

// 2. Load Go's wasm_exec.js runtime
const require = createRequire(import.meta.url);
require(join(__dirname, 'wasm_exec.js'));

// ========================= MAIN =========================

async function main() {
  // 3. Load and run Go WASM (cmd/init)
  const go = new globalThis.Go();
  const { instance } = await WebAssembly.instantiate(
    readFileSync(join(__dirname, 'main.wasm')),
    go.importObject,
  );
  go.run(instance);

  // Wait for Go init to finish
  while (!globalThis.hackpad?.ready) {
    await new Promise(r => setTimeout(r, 10));
  }

  const { hackpad, fs: vfs, child_process } = globalThis;
  const O = vfs.constants;

  // ====== overlayMemFS ======
  console.log('\n=== overlayMemFS ===');
  vfs.mkdirSync('/tmp/mem', 0o755);
  await hackpad.overlayMemFS('/tmp/mem');

  const fd1 = vfs.openSync('/tmp/mem/hello.txt', O.O_WRONLY | O.O_CREAT | O.O_TRUNC, 0o644);
  const msg = new TextEncoder().encode('Hello from memfs!');
  vfs.writeSync(fd1, msg, 0, msg.length, 0);
  vfs.closeSync(fd1);

  const fd2 = vfs.openSync('/tmp/mem/hello.txt', O.O_RDONLY, 0);
  const buf = new Uint8Array(64);
  const n = vfs.readSync(fd2, buf, 0, 64, 0);
  vfs.closeSync(fd2);
  console.log('read:', new TextDecoder().decode(buf.subarray(0, n)));

  // ====== overlayNodefs ======
  console.log('\n=== overlayNodefs ===');
  vfs.mkdirSync('/mnt', 0o755);
  vfs.mkdirSync('/mnt/host', 0o755);
  await hackpad.overlayNodefs('/mnt/host', __dirname, 'read');

  const entries = vfs.readdirSync('/mnt/host');
  // console.log('host dir:', entries.filter(e => e.endsWith('.mjs') || e.endsWith('.wasm')));
  console.log('host dir:', entries);

  const fd3 = vfs.openSync('/mnt/host/run.mjs', O.O_RDONLY, 0);
  const buf2 = new Uint8Array(80);
  const n2 = vfs.readSync(fd3, buf2, 0, 80, 0);
  vfs.closeSync(fd3);
  console.log('host file:', new TextDecoder().decode(buf2.subarray(0, n2)));

  // ====== spawn Go WASM child process ======
  console.log('\n=== Spawning child Go WASM ===');

  // Copy info.wasm into VFS with executable permission
  const wasmBuf = readFileSync(join(__dirname, 'info.wasm'));
  const fd4 = vfs.openSync('/tmp/info.wasm', O.O_WRONLY | O.O_CREAT | O.O_TRUNC, 0o755);
  vfs.writeSync(fd4, new Uint8Array(wasmBuf), 0, wasmBuf.length, 0);
  vfs.closeSync(fd4);

  // Verify it's executable
  const stat = vfs.statSync('/tmp/info.wasm');
  const isExec = (stat.mode & 0o111) !== 0;
  console.log('info.wasm executable:', isExec, 'mode:', stat.mode.toString(8));

  // Spawn it — stdout/stderr inherit to let output appear here
  const subprocess = child_process.spawn('/tmp/info.wasm', ['hello', 'from', 'outer', 'space'], {
    stdio: ['inherit', 'inherit', 'inherit'],
    cwd: '/tmp',
    env: {
      PATH: '/bin:/usr/local/go/bin:/home/me/go/bin',
      HOME: '/home/me',
      USER: 'me',
    },
  });
  await new Promise(r => setTimeout(r, 600)); // wait for inherited stdout to flush
  console.log('child pid:', subprocess.pid);

  // Wait for it
  const result = child_process.waitSync(subprocess.pid);
  console.log('exit result:', JSON.stringify(result));

  // ====== Mounts ======
  console.log('\n=== Mounts ===');
  console.log(hackpad.getMounts());

  console.log('\nAll OK — exiting');
  // Give child stdout pipes a moment to flush, then exit
  setTimeout(() => process.exit(0), 500);
}

main().catch(err => {
  console.error('FAILED:', err);
  process.exit(1);
});
