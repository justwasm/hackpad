import './webAssembly.js';
import { CDN } from './w9y.js';

const Go = window.Go;

let overlayProgress = 0;
let progressListeners = [];

async function init() {
  const startTime = new Date().getTime()
  const go = new Go();
  const cmd = await WebAssembly.instantiateStreaming(fetch(CDN.kernel), go.importObject)
  go.env = {
    'TERM': 'xterm-256color',
    'COLORTERM': 'truecolor',
    'CLICOLOR_FORCE': '1',
    'CRUSH_CORE_UTILS': '1',
    'CRUSH_DISABLE_PROVIDER_AUTO_UPDATE': '1',
    'DO_NOT_TRACK': '1',
    'GOMODCACHE': '/home/me/.cache/go-mod',
    'GOPROXY': 'https://goproxy.up.railway.app',
    'GONOSUMDB': '*',
    'HOME': '/home/me',
    'PATH': '/bin:/home/me/wanix:/home/me/go/bin',
    'USER': 'me',
    'WANIX': '/home/me/wanix',
    'USER': 'me',
  }
  go.run(cmd.instance)
  const { hackpad, fs } = window
  console.debug(`hackpad status: ${hackpad.ready ? 'ready' : 'not ready'}`)

  const mkdir = promisify(fs.mkdir)
  await mkdir("/bin", {mode: 0o700})
  await hackpad.overlayOPFS('/bin')
  await hackpad.overlayOPFS('/home/me')
  await mkdir("/home/me/.cache", {recursive: true, mode: 0o700})
  await hackpad.overlayOPFS('/home/me/.cache')

  console.debug("Startup took", (new Date().getTime() - startTime) / 1000, "seconds")
}

const initOnce = init();

export async function install(url, name) {
  await initOnce
  return window.hackpad.install(url, name)
}

export async function run(name, ...args) {
  const process = await spawn({ name, args })
  return await wait(process.pid)
}

export async function wait(pid) {
  await initOnce
  const { child_process } = window
  return await new Promise((resolve, reject) => {
    child_process.wait(pid, (err, process) => {
      if (err) {
        reject(err)
        return
      }
      resolve(process)
    })
  })
}

export async function spawn({ name, args, ...options }) {
  await initOnce
  const { child_process } = window
  return await new Promise((resolve, reject) => {
    const subprocess = child_process.spawn(name, args, options)
    if (subprocess.error) {
      reject(new Error(`Failed to spawn command: ${name} ${args.join(" ")}: ${subprocess.error}`))
      return
    }
    resolve(subprocess)
  })
}

export async function mkdirAll(path) {
  await initOnce
  const { fs } = window
  fs.mkdirSync(path, { recursive: true, mode: 0o755 })
}

export function observeGoDownloadProgress(callback) {
  progressListeners.push(callback)
  callback(overlayProgress)
}

const promisify = (fn) => (...args) => {
  return new Promise((resolve, reject) => {
    const newArgs = [...args]
    newArgs.push((err, ...results) => {
      if (err) {
        reject(err)
      } else {
        resolve(results)
      }
    })
    fn(...newArgs)
  })
}

/**
 * Mount a local directory via the File System Access API.
 * Shows a directory picker, then mounts the selected directory at the given path.
 *
 * @param {string} mountPath - Virtual filesystem path to mount at (e.g., "/home/me/project")
 * @param {Object} [options] - Options
 * @param {boolean} [options.writable=false] - Request write access (default: read-only)
 * @param {string} [options.id] - Unique ID to persist directory access across sessions
 * @param {string} [options.startIn] - Suggested starting directory for the picker
 * @returns {Promise<{name: string}>} Resolves with the picked directory name
 */
export async function mountLocalDir(mountPath, options = {}) {
  await initOnce
  const pickerOpts = { mode: options.writable ? 'readwrite' : 'read' }
  if (options.id) pickerOpts.id = options.id
  if (options.startIn) pickerOpts.startIn = options.startIn

  const handle = await window.showDirectoryPicker(pickerOpts)
  const mode = options.writable ? 'readwrite' : 'read'

  // Ensure the mount directory exists
  try {
    await mkdirAll(mountPath)
  } catch (e) {
    // Might already exist, that's ok
  }

  // Pass the handle to Go WASM to mount
  await promisify(window.hackpad.overlayLocalDir)(mountPath, handle, mode)

  console.debug(`Mounted local directory "${handle.name}" at ${mountPath} (mode: ${mode})`)
  return { name: handle.name }
}
