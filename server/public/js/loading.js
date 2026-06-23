import { html } from './html.js';

export function Loading({ percentage }) {
  return html`<div className="app-loading">
    <div className="app-loading-center">
      <div className="app-loading-spinner">
        <img className="app-loading-logo" src="icon.png" alt="" />
      </div>
      <p>
        installing <span className="app-title">go4js</span>
      </p>
      <div className="app-loading-bar">
        <div className="app-loading-bar-fill" style=${{ width: `${percentage || 0}%` }} />
      </div>
      ${percentage !== undefined ?
        html`<p className="app-loading-bar-text">${Math.round(percentage)}%</p>`
      : null}
    </div>
  </div>`
}
