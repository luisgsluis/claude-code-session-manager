// Applies the saved skin before first paint (data-skin on <html>), so
// switching to a non-default skin doesn't flash the dark default first.
// External file (not inline) because the CSP's script-src is 'self'
// 'unsafe-eval' only, no 'unsafe-inline' — an inline <script> is silently
// blocked by the browser. app.js's initSkin() re-applies this after Alpine
// loads, so a blocked/cached-stale copy of this file can't leave the skin
// unapplied.
(function () {
  try {
    var s = localStorage.getItem('ccsm_skin');
    if (s === 'light' || s === 'contrast' || s === 'solarized') document.documentElement.setAttribute('data-skin', s);
  } catch (e) { /* private mode */ }
})();
