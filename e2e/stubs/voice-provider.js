#!/usr/bin/env node
// Stub of an OpenAI-compatible provider (what Groq and DeepSeek both are), so
// the e2e suite can exercise dictation and rewriting without a network call,
// an API key, or a bill.
//
// It is deliberately scriptable through the text it receives, which is what
// lets one stub cover every branch the UI has to handle:
//
//   "AMBIGUO"  -> the rewrite asks two questions, one per round: an options
//                 question first, then a free-text one, then stops
//   "LARGO"    -> the rewrite comes back over the send limit
//   "FALLO"    -> the provider answers 500, so the UI must surface an error
//   "BASURA"   -> the model answers prose instead of JSON (the tolerant parser)
//
// Everything else gets a normal rewrite that echoes what it was given, so a
// test can assert the transcription actually reached the rewriter.

const http = require('http');

const PORT = Number(process.env.VOICE_STUB_PORT || 8797);

// Fixed transcription. Real speech-to-text of a fake audio device would return
// nothing useful, and the point of the test is the plumbing, not the acoustics.
const TRANSCRIPT = process.env.VOICE_STUB_TRANSCRIPT ||
  'pues eh mira el sonar ese que no importa los episodios bueno el sonarr';

function readBody(req) {
  return new Promise((resolve) => {
    const chunks = [];
    req.on('data', (c) => chunks.push(c));
    req.on('end', () => resolve(Buffer.concat(chunks)));
  });
}

function chatReply(content) {
  return JSON.stringify({ choices: [{ message: { content } }] });
}

const server = http.createServer(async (req, res) => {
  const body = await readBody(req);
  res.setHeader('Content-Type', 'application/json');

  // Readiness probe for Playwright's webServer.
  if (req.url === '/health') { res.end(JSON.stringify({ ok: true })); return; }

  // Record what arrived so a test can assert on the request CCSM built.
  const auth = req.headers['authorization'] || '';

  if (req.url.endsWith('/audio/transcriptions')) {
    // The audio really is uploaded: assert it is not empty, so a broken
    // MediaRecorder wiring fails loudly here rather than silently transcribing
    // nothing.
    if (body.length === 0) {
      res.statusCode = 400;
      res.end(JSON.stringify({ error: 'empty upload' }));
      return;
    }
    if (!auth.startsWith('Bearer ')) {
      res.statusCode = 401;
      res.end(JSON.stringify({ error: 'no credential' }));
      return;
    }
    res.end(JSON.stringify({ text: TRANSCRIPT, _bytes: body.length }));
    return;
  }

  if (req.url.endsWith('/chat/completions')) {
    let payload = {};
    try { payload = JSON.parse(body.toString()); } catch (e) { /* handled below */ }
    const messages = payload.messages || [];
    const system = (messages[0] && messages[0].content) || '';
    const user = (messages[1] && messages[1].content) || '';

    if (user.includes('FALLO')) {
      res.statusCode = 500;
      res.end(JSON.stringify({ error: 'provider exploded, and here is the key: ' + auth }));
      return;
    }
    if (user.includes('BASURA')) {
      res.end(chatReply('Sure! Here is what you asked for:\n' +
        JSON.stringify({ role: 'devops', question: null, prompt: 'RESCATADO DE PROSA' }) +
        '\nHope that helps.'));
      return;
    }
    if (user.includes('LARGO')) {
      res.end(chatReply(JSON.stringify({
        role: 'devops', question: null, prompt: 'X'.repeat(20000),
      })));
      return;
    }
    // AMBIGUO walks the sequential clarification loop: one question per
    // round, counted from how many "- Q:" answer lines the client folded
    // into the request so far, so the stub proves the UI actually loops
    // (asks again after an answer) instead of only handling a single round.
    if (user.includes('AMBIGUO')) {
      const round = (user.match(/^- Q:/gm) || []).length;
      if (round === 0) {
        res.end(chatReply(JSON.stringify({
          role: 'devops',
          question: { text: '¿El sonarr de la Pi o el del NAS?', options: ['el de la Pi', 'el del NAS'] },
          prompt: 'PROMPT PROVISIONAL sobre sonarr',
        })));
        return;
      }
      if (round === 1) {
        res.end(chatReply(JSON.stringify({
          role: 'devops',
          question: { text: '¿Qué episodio concreto?' },
          prompt: 'PROMPT PROVISIONAL sobre sonarr (destino confirmado)',
        })));
        return;
      }
      res.end(chatReply(JSON.stringify({
        role: 'devops', question: null,
        prompt: 'PROMPT FINAL sobre sonarr :: ' + user.replace(/\s+/g, ' ').trim().slice(0, 300),
      })));
      return;
    }

    // Report which role blocks the server actually sent, so a test can prove
    // that a forced role does not leak the other roles' instructions and that
    // auto mode shows the model all of them. It rides in the prompt text
    // because an invented role id would be replaced by the server's fallback.
    const hasDocs = system.includes('Rewrite it as a documentation request');
    const hasDevops = system.includes('Rewrite it as an operations request');
    let blocks = 'BLOQUES:ninguno';
    if (hasDocs && hasDevops) blocks = 'BLOQUES:todos';
    else if (hasDocs) blocks = 'BLOQUES:docs';
    else if (hasDevops) blocks = 'BLOQUES:devops';

    res.end(chatReply(JSON.stringify({
      role: hasDocs && !hasDevops ? 'docs' : 'devops',
      question: null,
      prompt: 'PROMPT REESCRITO :: ' + blocks + ' :: ' +
        user.replace(/\s+/g, ' ').trim().slice(0, 300),
    })));
    return;
  }

  res.statusCode = 404;
  res.end(JSON.stringify({ error: 'not found' }));
});

server.listen(PORT, '127.0.0.1', () => {
  process.stdout.write('voice stub on ' + PORT + '\n');
});
