import MobileDetect from 'mobile-detect';
import { html } from './html.js';

const md = new MobileDetect(window.navigator.userAgent);
let browserName = ""
if (window.navigator.vendor.match(/google/i)) {
  browserName = 'Chrome'
} else if (navigator.userAgent.match(/firefox\//i)) {
  browserName = 'Firefox'
}
const knownWorkingBrowsers = [
  'Chrome',
  'Firefox',
]
const isCompatibleBrowser = md.mobile() === null && knownWorkingBrowsers.includes(browserName)

function joinOr(arr) {
  if (arr.length === 1) {
    return arr[0]
  }
  if (arr.length === 2) {
    return `${arr[0]} or ${arr[1]}`
  }
  const commaDelimited = arr.slice(0, arr.length - 1).join(", ")
  return `${commaDelimited}, or ${arr[arr.length - 1]}`
}

export function Compat() {
  if (isCompatibleBrowser) {
    return null
  }
  return html`<div className="compat">
    <p>go4js may not work reliably in your browser.</p>
    <p>If you're experience any issues, try a recent version of ${joinOr(knownWorkingBrowsers)} on a device with enough memory, like a PC.</p>
  </div>`
}
