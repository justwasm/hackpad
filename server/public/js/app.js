import React from 'react';
import { createRoot } from 'react-dom/client';
import { html } from './html.js';

import { install, run, observeGoDownloadProgress } from './hackpad.js';
import { newEditor, setupMonaco } from './editor.js';
import { newTerminal } from './terminal.js';
import { Compat } from './compat.js';
import { Loading } from './loading.js';
import { CDN } from './w9y.js';

function App() {
  const [percentage, setPercentage] = React.useState(0);
  const [loading, setLoading] = React.useState(true);
  React.useEffect(() => {
    observeGoDownloadProgress(setPercentage)

    Promise.all([
      import('monaco-editor').then(monaco => {
        setupMonaco(monaco);
        window.editor = {
          newTerminal,
          newEditor: (elem, onEdit) => newEditor(elem, onEdit, monaco),
        };
      }),
      install(CDN.editor, 'editor'),
      install(CDN.sh, 'sh'),
    ]).then(() => {
      run('editor', '--editor=editor')
      setLoading(false)
    })
  }, [])

  return html`<${React.Fragment}>
      ${loading ? html`<${React.Fragment}>
        <${Compat} />
        <${Loading} percentage=${percentage} />
      </${React.Fragment}>` : null}
      <div id="app">
        <div id="editor"></div>
      </div>
    </${React.Fragment}>`;
}

const root = createRoot(document.getElementById('root'));
root.render(html`<${React.StrictMode}><${App} /></${React.StrictMode}>`);
