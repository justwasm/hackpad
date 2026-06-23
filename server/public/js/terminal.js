import { Terminal as XTerminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { ImageAddon } from '@xterm/addon-image';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { ClipboardAddon } from '@xterm/addon-clipboard';
import { listenColorScheme } from './colorScheme.js';

const fontScale = 0.85

export function newTerminal(elem) {
  const fitAddon = new FitAddon()
  const imageAddon = new ImageAddon()
  const term = new XTerminal({
    convertEol: true,
    cursorBlink: true,
    allowTransparency: true,
    scrollbar: { showScrollbar: false },
  })
  term.loadAddon(fitAddon)
  term.loadAddon(imageAddon)
  term.loadAddon(new WebLinksAddon())
  term.loadAddon(new ClipboardAddon())

  const dark = "rgb(33, 33, 33)"
  const light = "white"
  listenColorScheme({
    light: () => { term.options.theme = {
      background: light,
      foreground: dark,
      cursor: dark,
      selectionBackground: 'rgba(0, 0, 0, 0.2)',
      selectionInactiveBackground: 'rgba(0, 0, 0, 0.1)',
    } },
    dark: () => { term.options.theme = {
      background: dark,
      foreground: light,
      cursor: light,
      selectionBackground: 'rgba(255, 255, 255, 0.25)',
      selectionInactiveBackground: 'rgba(255, 255, 255, 0.12)',
    } },
  })

  term.open(elem)
  term.options.cursorBlink = true
  term.focus()
  term.parser.registerOscHandler(0, (data) => {
    if (typeof term.__onXtermTitle === 'function') {
      term.__onXtermTitle(data)
    }
    return true
  })
  const fit = () => {
    const fontSize = parseFloat(getComputedStyle(elem).fontSize)
    term.options.fontSize = fontSize * fontScale
    fitAddon.fit()
    const termID = elem.dataset?.termid || ''
    window.hackpad?.setWinch?.(termID, term.cols, term.rows, elem.offsetWidth, elem.offsetHeight)
  }

  fit()
  elem._onTabFocus = fit
  if (window.ResizeObserver) {
    const observer = new ResizeObserver(() => {
      if (!elem.parentNode) {
        observer.disconnect()
        return
      }
      if (elem.classList.contains("active")) {
        fit()
      }
    })
    observer.observe(elem.parentNode)
  } else {
    window.addEventListener('resize', fit)
  }
  return term
}
