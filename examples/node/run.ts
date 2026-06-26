#!/usr/bin/env -S deno run -A --unstable-raw-imports
// Example: running hackpad WASM in Deno with overlayNodefs + overlayMemFS
//
// Build:
//   PATH=cache/go/bin:cache/go/misc/wasm:$PATH GOOS=js GOARCH=wasm \
//     go build -o examples/node/main.wasm ./cmd/init/
//   PATH=cache/go/bin:cache/go/misc/wasm:$PATH GOOS=js GOARCH=wasm \
//     go build -o examples/node/info.wasm ./examples/node/info/
//   cp server/public/wasm/wasm_exec.js examples/node/
//
// Run:
//   deno run -A examples/node/run.ts

import mainWasm from './main.wasm' with { type: 'bytes' };
import infoWasm from './info.wasm' with { type: 'bytes' };
import './wasm_exec.js';

// ---- 1. Inject __nodefs bridge (required by overlayNodefs) ----
globalThis.__nodefs = (() => {
  const statToObj = (s: Deno.FileInfo) => ({
    dev: 0, ino: 0, mode: s.mode ?? 0, nlink: 0,
    uid: 0, gid: 0, rdev: 0, size: s.size,
    blksize: 0, blocks: 0,
    isDirectory: s.isDirectory, isFile: s.isFile,
    isSymbolicLink: s.isSymlink,
    atimeMs: s.atime?.getTime() ?? 0,
    mtimeMs: s.mtime?.getTime() ?? 0,
  });

  return {
    readFileSync(p: string) {
      return { data: Deno.readFileSync(p).buffer, err: null };
    },
    writeFileSync(p: string, b: Uint8Array) {
      Deno.writeFileSync(p, new Uint8Array(b)); return { err: null };
    },
    statSync(p: string) {
      return { stat: statToObj(Deno.statSync(p)), err: null };
    },
    lstatSync(p: string) {
      return { stat: statToObj(Deno.lstatSync(p)), err: null };
    },
    mkdirSync(p: string, m: number) {
      Deno.mkdirSync(p, { mode: m }); return { err: null };
    },
    mkdirAllSync(p: string, m: number) {
      Deno.mkdirSync(p, { recursive: true, mode: m }); return { err: null };
    },
    readdirSync(p: string) {
      const entries: { name: string; isDirectory: boolean; isSymbolicLink: boolean }[] = [];
      for (const e of Deno.readDirSync(p)) {
        entries.push({ name: e.name, isDirectory: e.isDirectory, isSymbolicLink: e.isSymlink });
      }
      return { entries, err: null };
    },
    unlinkSync(p: string) { Deno.removeSync(p); return { err: null }; },
    rmdirSync(p: string) { Deno.removeSync(p); return { err: null }; },
    rmSync(p: string, r: boolean) { Deno.removeSync(p, { recursive: r }); return { err: null }; },
    renameSync(o: string, n: string) { Deno.renameSync(o, n); return { err: null }; },
    chmodSync(p: string, m: number) { Deno.chmodSync(p, m); return { err: null }; },
    utimesSync(p: string, a: number, m: number) { Deno.utimeSync(p, a, m); return { err: null }; },
    symlinkSync(t: string, p: string) { Deno.symlinkSync(t, p); return { err: null }; },
    readlinkSync(p: string) { return { link: Deno.readLinkSync(p), err: null }; },
    accessSync(p: string, _m: number) {
      try { Deno.statSync(p); return { err: null }; }
      catch (e: unknown) { return { err: e instanceof Error ? e.message : String(e) }; }
    },
  };
})();

// ---- 2. Main ----
async function main() {
  const go = new (globalThis as Record<string, unknown>).Go() as {
    argv: string[];
    env: Record<string, string>;
    importObject: WebAssembly.Imports;
    run(instance: WebAssembly.Instance): Promise<void>;
  };
  go.argv = ['js'];

  const wasmBytes = mainWasm;
  const { instance } = await WebAssembly.instantiate(wasmBytes, go.importObject);
  go.run(instance);

  while (!(globalThis as Record<string, unknown>).hackpad?.ready) {
    await new Promise(r => setTimeout(r, 10));
  }

  const hackpad = (globalThis as Record<string, unknown>).hackpad as Record<string, unknown>;
  const vfs = (globalThis as Record<string, unknown>).fs as Record<string, unknown>;
  const child_process = (globalThis as Record<string, unknown>).child_process as Record<string, unknown>;
  const O = (vfs as Record<string, Record<string, number>>).constants;

  // ====== overlayMemFS ======
  console.log('\n=== overlayMemFS ===');
  await cb(vfs.mkdir as CallbackFn, '/tmp/mem', 0o755);
  await (hackpad.overlayMemFS as (...args: unknown[]) => Promise<unknown>)('/tmp/mem');

  const fd1 = (vfs.openSync as (...args: unknown[]) => number)(
    '/tmp/mem/hello.txt', O.O_WRONLY | O.O_CREAT | O.O_TRUNC, 0o644,
  );
  const msg = new TextEncoder().encode('Hello from memfs!');
  (vfs.writeSync as (...args: unknown[]) => number)(fd1, msg, 0, msg.length, 0);
  (vfs.closeSync as (fd: number) => void)(fd1);

  const fd2 = (vfs.openSync as (...args: unknown[]) => number)('/tmp/mem/hello.txt', O.O_RDONLY, 0);
  const buf = new Uint8Array(64);
  const n = (vfs.readSync as (...args: unknown[]) => number)(fd2, buf, 0, 64, 0);
  (vfs.closeSync as (fd: number) => void)(fd2);
  console.log('read:', new TextDecoder().decode(buf.subarray(0, n)));

  // ====== overlayNodefs ======
  console.log('\n=== overlayNodefs ===');
  await cb(vfs.mkdir as CallbackFn, '/mnt', 0o755);
  await cb(vfs.mkdir as CallbackFn, '/mnt/host', 0o755);
  await (hackpad.overlayNodefs as (...args: unknown[]) => Promise<unknown>)(
    '/mnt/host', import.meta.dirname!, 'read',
  );

  const entries = await cb(vfs.readdir as CallbackFn, '/mnt/host') as string[];
  console.log('host dir:', entries.filter(e => e.endsWith('.mjs') || e.endsWith('.wasm') || e.endsWith('.ts')));

  const fd3 = (vfs.openSync as (...args: unknown[]) => number)('/mnt/host/run.ts', O.O_RDONLY, 0);
  const buf2 = new Uint8Array(80);
  const n2 = (vfs.readSync as (...args: unknown[]) => number)(fd3, buf2, 0, 80, 0);
  (vfs.closeSync as (fd: number) => void)(fd3);
  console.log('host file:', new TextDecoder().decode(buf2.subarray(0, n2)));

  // ====== spawn Go WASM child ======
  console.log('\n=== Spawning child Go WASM ===');

  const fd4 = (vfs.openSync as (...args: unknown[]) => number)(
    '/tmp/info.wasm', O.O_WRONLY | O.O_CREAT | O.O_TRUNC, 0o755,
  );
  (vfs.writeSync as (...args: unknown[]) => number)(fd4, infoWasm, 0, infoWasm.length, 0);
  (vfs.closeSync as (fd: number) => void)(fd4);

  const subprocess = (child_process.spawn as (...args: unknown[]) => Record<string, unknown>)(
    '/tmp/info.wasm',
    ['hello', 'from', 'deno'],
    {
      stdio: ['inherit', 'inherit', 'inherit'],
      cwd: '/tmp',
      env: {
        PATH: '/bin:/usr/local/go/bin:/home/me/go/bin',
        HOME: '/home/me',
        USER: 'me',
      },
    },
  );
  await new Promise(r => setTimeout(r, 600)); // wait for inherited stdout to flush
  console.log('child pid:', subprocess.pid);

  const result = await cb(child_process.wait as CallbackFn, subprocess.pid);
  console.log('exit result:', JSON.stringify(result));

  // ====== Mounts ======
  console.log('\n=== Mounts ===');
  console.log((hackpad.getMounts as () => string[])());

  console.log('\nAll OK — exiting');
  setTimeout(() => Deno.exit(0), 500);
}

type CallbackFn = (...args: unknown[]) => void;

function cb(fn: CallbackFn, ...args: unknown[]): Promise<unknown> {
  return new Promise((resolve, reject) => {
    fn(...args, (err: unknown, ...results: unknown[]) => {
      if (err) reject(err);
      else resolve(results.length <= 1 ? results[0] : results);
    });
  });
}

main().catch(err => {
  console.error('FAILED:', err);
  Deno.exit(1);
});
