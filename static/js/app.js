// Translations for the CCSM UI. es is the default; en is the alternative.
const I18N = {
  es: {
    logout: 'salir',
    refresh: 'Refrescar',
    new_session: 'Nueva sesión',
    new_quick: 'Nueva sesión rápida',
    new_advanced: 'Nueva sesión avanzada',
    adv_tmux: 'Nombre sesión tmux',
    adv_tmux_ph: 'opcional',
    adv_claude: 'Nombre sesión Claude',
    adv_claude_ph: 'opcional',
    adv_profile: 'Perfil a aplicar',
    adv_profile_default: 'Perfil activo (por defecto)',
    adv_project: 'Proyecto (CLAUDE.md)',
    adv_project_default: 'Principal (inicio)',
    adv_create: 'Crear sesión',
    new_with_profile: 'Nueva con perfil',
    change_profile: 'Cambiar perfil',
    pick_profile: 'Elige un perfil para la sesión nueva:',
    pick_profile_apply: 'Perfil que se aplicará a las próximas sesiones:',
    active: 'activo',
    no_profiles: 'No hay perfiles',
    working: 'trabajando...',
    active_sessions: 'Sesiones activas',
    loading: 'cargando...',
    no_sessions: 'No hay sesiones activas',
    no_sessions_sub: 'Crea una nueva sesión o retoma una conversación',
    session_label: 'sesión {0}',
    no_task: '(sin tarea)',
    copy_attach: 'Copiar comando attach',
    close_session: 'Cerrar sesión',
    conversations: 'Conversaciones',
    switch_cards: '▦ tarjetas',
    switch_list: '≡ lista',
    search_conv: 'Buscar conversación...',
    searching: 'buscando...',
    no_conv: 'No se encontraron conversaciones',
    no_conv_search: 'Prueba con otros términos',
    no_conv_empty: 'Crea una sesión nueva para empezar',
    conv_origin_all: 'origen: todos',
    conv_from: 'Desde esta fecha',
    conv_to: 'Hasta esta fecha',
    conv_alive: 'solo vivas',
    export_conv: 'Descargar conversación (txt)',
    pin: 'Fijar',
    unpin: 'Quitar fijada',
    archive: 'Archivar',
    unarchive: 'Desarchivar',
    meta_save: 'Guardar etiquetas/notas',
    meta_saved: 'Guardado.',
    tags_notes_label: 'Etiquetas / Notas',
    view_profile_title: 'ver perfil {0}',
    profile_label: 'Perfil: {0}',
    metrics_all_models: 'Todos los modelos',
    tags_placeholder: 'etiquetas separadas por coma',
    notes_placeholder: 'Notas (máx. 500 caracteres)',
    empty_conv: '(vacía)',
    alive: 'sesión viva',
    resume: 'retomar',
    preview: 'preview',
    load_more: 'cargar más',
    preview_title: 'Preview de conversación',
    you: '🧑 Tú',
    claude: '🤖 Claude',
    login_subtitle: 'Inicia sesión para continuar',
    username: 'Usuario',
    password: 'Contraseña',
    login_button: 'Entrar',
    cfg_title: 'Configuración',
    cfg_deploy: 'Despliegue',
    cfg_mode: 'modo',
    cfg_port: 'puerto',
    cfg_attach: 'host attach',
    cfg_paths: 'Rutas',
    cfg_security: 'Seguridad',
    cfg_lan: 'redes LAN',
    cfg_socket: 'socket agente',
    cfg_users: 'usuarios',
    cfg_rc: 'Bootstrap RC',
    cfg_rc_bootstrap: 'perfil bootstrap',
    cfg_rc_wait: 'espera',
    cfg_rc_poll: 'sondeo',
    cfg_note: 'Para cambiar estos valores edita config.yaml (o las variables CCSM_*) y reinicia el servicio.',
    cfg_secrets_note: 'Los secretos (session_secret, agent_secret y las contraseñas) no se muestran por seguridad.',
    cfg_copy: 'Copiar',
    cfg_view_settings: 'Ver settings.json actual',
    cfg_viewing_settings: 'settings.json actual',
    cfg_direct: 'modo directo (sin agente)',
    toast_session_created: 'Sesión {0} creada',
    toast_session_resumed: 'Sesión {0} retomada',
    toast_session_killed: 'Sesión {0} cerrada',
    toast_profile_applied: 'Perfil {0} aplicado',
    toast_attach_copied: 'Comando copiado: {0}',
    toast_attach_fallback: 'Comando: {0}',
    toast_error_meta: 'Error al guardar etiquetas/notas',
    toast_error_create: 'Error al crear sesión',
    toast_error_resume: 'Error al retomar',
    toast_error_conn: 'Error: {0}',
    toast_error_auth: 'Error de autenticación',
    toast_error_conn_login: 'Error de conexión',
    confirm_kill: '¿Cerrar sesión {0}?',
    lan_label: '[lan]',
    name_placeholder: 'Nombre (opcional)',
    name_invalid: 'A-Z, a-z, 0-9, _-, 32 car.',
    rename: 'Renombrar sesión',
    rename_title: 'Renombrar {0}',
    rename_tmux: 'Nombre tmux',
    rename_claude: 'Nombre Claude',
    rename_tmux_apply: 'Renombrar tmux',
    rename_claude_apply: 'Renombrar Claude',
    toast_session_renamed: 'Sesión tmux renombrada a {0}',
    toast_claude_renamed: 'Sesión Claude renombrada',
    cfg_edit: 'Editar',
    cfg_save: 'Guardar',
    cfg_cancel: 'Cancelar',
    cfg_restart_note: 'Cambios en port, agent_socket o paths requieren reinicio del servicio (systemctl restart ccsm).',
    cfg_restart_needed: '\u26a0\ufe0f requiere reinicio',
    cfg_add_user: '+ A\u00f1adir usuario',
    cfg_del_user: 'Eliminar',
    cfg_chg_pass: 'Cambiar clave',
    cfg_add_user_title: 'A\u00f1adir usuario',
    cfg_chg_pass_title: 'Cambiar contrase\u00f1a de {0}',
    cfg_password_label: 'Contrase\u00f1a (m\u00edn. 8 caracteres)',
    cfg_confirm_del_user: '\u00bfEliminar usuario {0}?',
    cfg_last_user: 'No se puede eliminar el \u00faltimo usuario.',
    cfg_user_added: 'Usuario {0} creado.',
    cfg_user_deleted: 'Usuario {0} eliminado.',
    cfg_password_changed: 'Contrase\u00f1a cambiada.',
    cfg_saved: 'Guardado.',
    cfg_close: 'Cerrar',
    audit_title: 'Registro de actividad',
    audit_filter: 'Filtrar (sesi\u00f3n, perfil, usuario, acci\u00f3n)\u2026',
    audit_empty: 'Sin eventos registrados.',
    metrics_title: 'M\u00e9tricas',
    metrics_uptime: 'Activo',
    metrics_ram: 'RAM',
    metrics_sessions_day: 'Sesiones por d\u00eda',
    metrics_profiles: 'Perfiles m\u00e1s usados',
    metrics_tokens: 'Tokens por d\u00eda (entrada/salida/cach\u00e9)',
    metrics_input: 'entrada',
    metrics_output: 'salida',
    metrics_cache: 'cach\u00e9',
    metrics_none: 'Sin datos a\u00fan.',
    metrics_by_model: 'Tokens por modelo',
    metrics_download_json: 'Descargar JSON',
    metrics_download_csv: 'Descargar CSV',
    notify_unsupported: 'El navegador no soporta notificaciones.',
    notify_grant: 'Activar notificaciones',
    notify_denied: 'Notificaciones denegadas en el navegador.',
    notify_unmuted: 'Notificaciones activadas',
    notify_muted: 'Notificaciones silenciadas',
    notify_action_login: 'Inicio de sesión',
    notify_action_session_new: 'Nueva sesi\u00f3n',
    notify_action_session_kill: 'Sesi\u00f3n cerrada',
    notify_action_session_resume: 'Sesi\u00f3n retomada',
    notify_action_profile_apply: 'Perfil aplicado',
    notify_action_turn_complete: 'Turno completado',
    notify_action_session_waiting: 'Esperando aprobaci\u00f3n',
    notify_action_session_choice: 'Necesita tu decisi\u00f3n',
    live_view: 'Ver sesi\u00f3n en vivo',
    live_title: 'Sesi\u00f3n {0} en vivo',
    live_closed: 'Sesi\u00f3n cerrada o desconectada.',
    live_reconnecting: 'Conexi\u00f3n perdida, reconectando\u2026',
    live_tab_chat: 'Chat',
    live_tab_term: 'Terminal',
    live_scroll_up: 'Subir',
    live_scroll_end: 'Ir al final',
    live_screen_sep: '──── Pantalla actual ────',
    mode_label: 'Modo',
    model_label: 'Modelo',
    chat_no_conv: 'A\u00fan no hay conversaci\u00f3n (esperando el primer mensaje\u2026)',
    chat_placeholder: 'Escribe un mensaje\u2026 (Enter env\u00eda, Shift+Enter salto de l\u00ednea)',
    chat_send: 'Enviar',
    chat_err: 'No se pudo enviar el mensaje',
    chat_status_alive: 'En vivo',
    chat_status_off: 'Desconectada',
    chat_status_waiting: 'esperando aprobación',
    chat_status_choice: 'eligiendo opción',
    chat_approve: 'Aprobar',
    chat_approve_title: 'Aprobar el comando (opción 1)',
    chat_choice_title: 'Elegir opción (Enter)',
    chat_choice_up: 'Opción anterior',
    chat_choice_down: 'Siguiente opción',
    chat_choice_hint: '↑/↓ para mover, Enter o clic para elegir',
    chat_stop_title: 'Cancelar / Escape',
    chat_status_mode: 'modo: {0}',
    chat_status_model: 'modelo: {0}',
    chat_status_elapsed: '{0}',
    rc_reconnect_on: 'RC: re-registrar',
    rc_reconnect_off: 'RC: activar',
    rc_reconnect_title: 'Reconectar Remote Control: fuerza un re-registro del bridge para que la sesión aparezca en la app de Claude.',
    confirm_rc_reconnect_on: 'Se re-registrará el Remote Control; si el bridge no se recupera en vivo, la sesión se cerrará y relanzará sola (retomando la conversación). ¿Continuar?',
    confirm_rc_reconnect_off: 'Se activará el Remote Control; si el bridge no se recupera en vivo, la sesión se cerrará y relanzará sola (retomando la conversación). ¿Continuar?',
    toast_rc_reconnect: 'Remote Control re-registrado',
    toast_rc_recovered: 'Bridge no recuperable en vivo: sesión relanzada como {0}',
    toast_rc_fail: 'No se pudo re-registrar el Remote Control ni relanzar la sesión ({0})',
    mode_sent: 'Modo cambiado a {0}',
    model_sent: 'Modelo cambiado a {0}',
  },
  en: {
    logout: 'logout',
    refresh: 'Refresh',
    new_session: 'New session',
    new_quick: 'Quick session',
    new_advanced: 'Advanced session',
    adv_tmux: 'tmux session name',
    adv_tmux_ph: 'optional',
    adv_claude: 'Claude session name',
    adv_claude_ph: 'optional',
    adv_profile: 'Profile to apply',
    adv_profile_default: 'Active profile (default)',
    adv_project: 'Project (CLAUDE.md)',
    adv_project_default: 'Principal (home)',
    adv_create: 'Create session',
    new_with_profile: 'New with profile',
    change_profile: 'Change profile',
    pick_profile: 'Pick a profile for the new session:',
    pick_profile_apply: 'Profile applied to upcoming sessions:',
    active: 'active',
    no_profiles: 'No profiles',
    working: 'working...',
    active_sessions: 'Active sessions',
    loading: 'loading...',
    no_sessions: 'No active sessions',
    no_sessions_sub: 'Start a new session or resume a conversation',
    session_label: 'session {0}',
    no_task: '(no task)',
    copy_attach: 'Copy attach command',
    close_session: 'Close session',
    conversations: 'Conversations',
    switch_cards: '▦ cards',
    switch_list: '≡ list',
    search_conv: 'Search conversation...',
    searching: 'searching...',
    no_conv: 'No conversations found',
    no_conv_search: 'Try different terms',
    no_conv_empty: 'Start a new session to get going',
    conv_origin_all: 'origin: all',
    conv_from: 'From this date',
    conv_to: 'Until this date',
    conv_alive: 'live only',
    export_conv: 'Download conversation (txt)',
    pin: 'Pin',
    unpin: 'Unpin',
    archive: 'Archive',
    unarchive: 'Unarchive',
    meta_save: 'Save tags/notes',
    meta_saved: 'Saved.',
    tags_notes_label: 'Tags / Notes',
    view_profile_title: 'view profile {0}',
    profile_label: 'Profile: {0}',
    metrics_all_models: 'All models',
    tags_placeholder: 'tags separated by commas',
    notes_placeholder: 'Notes (max 500 chars)',
    empty_conv: '(empty)',
    alive: 'live session',
    resume: 'resume',
    preview: 'preview',
    load_more: 'load more',
    preview_title: 'Conversation preview',
    you: '🧑 You',
    claude: '🤖 Claude',
    login_subtitle: 'Sign in to continue',
    username: 'Username',
    password: 'Password',
    login_button: 'Sign in',
    cfg_title: 'Settings',
    cfg_deploy: 'Deployment',
    cfg_mode: 'mode',
    cfg_port: 'port',
    cfg_attach: 'host attach',
    cfg_paths: 'Paths',
    cfg_security: 'Security',
    cfg_lan: 'LAN nets',
    cfg_socket: 'agent socket',
    cfg_users: 'users',
    cfg_rc: 'RC bootstrap',
    cfg_rc_bootstrap: 'bootstrap profile',
    cfg_rc_wait: 'wait',
    cfg_rc_poll: 'poll',
    cfg_note: 'To change these values edit config.yaml (or the CCSM_* env vars) and restart the service.',
    cfg_secrets_note: 'Secrets (session_secret, agent_secret and passwords) are never shown.',
    cfg_copy: 'Copy',
    cfg_view_settings: 'View current settings.json',
    cfg_viewing_settings: 'current settings.json',
    cfg_direct: 'direct mode (no agent)',
    toast_session_created: 'Session {0} created',
    toast_session_resumed: 'Session {0} resumed',
    toast_session_killed: 'Session {0} closed',
    toast_profile_applied: 'Profile {0} applied',
    toast_attach_copied: 'Command copied: {0}',
    toast_attach_fallback: 'Command: {0}',
    toast_error_meta: 'Error saving tags/notes',
    toast_error_create: 'Error creating session',
    toast_error_resume: 'Error resuming',
    toast_error_conn: 'Error: {0}',
    toast_error_auth: 'Authentication error',
    toast_error_conn_login: 'Connection error',
    confirm_kill: 'Close session {0}?',
    lan_label: '[lan]',
    name_placeholder: 'Name (optional)',
    name_invalid: 'A-Z, a-z, 0-9, _-, 32 chars',
    rename: 'Rename session',
    rename_title: 'Rename {0}',
    rename_tmux: 'tmux name',
    rename_claude: 'Claude name',
    rename_tmux_apply: 'Rename tmux',
    rename_claude_apply: 'Rename Claude',
    toast_session_renamed: 'tmux session renamed to {0}',
    toast_claude_renamed: 'Claude session renamed',
    cfg_edit: 'Edit',
    cfg_save: 'Save',
    cfg_cancel: 'Cancel',
    cfg_restart_note: 'Changes to port, agent_socket or paths need a service restart (systemctl restart ccsm).',
    cfg_restart_needed: '\u26a0\ufe0f restart required',
    cfg_add_user: '+ Add user',
    cfg_del_user: 'Delete',
    cfg_chg_pass: 'Change password',
    cfg_add_user_title: 'Add user',
    cfg_chg_pass_title: 'Change password for {0}',
    cfg_password_label: 'Password (min. 8 chars)',
    cfg_confirm_del_user: 'Delete user {0}?',
    cfg_last_user: 'Cannot delete the last user.',
    cfg_user_added: 'User {0} created.',
    cfg_user_deleted: 'User {0} deleted.',
    cfg_password_changed: 'Password changed.',
    cfg_saved: 'Saved.',
    cfg_close: 'Close',
    audit_title: 'Activity log',
    audit_filter: 'Filter (session, profile, user, action)…',
    audit_empty: 'No events recorded yet.',
    metrics_title: 'Metrics',
    metrics_uptime: 'Uptime',
    metrics_ram: 'RAM',
    metrics_sessions_day: 'Sessions per day',
    metrics_profiles: 'Most used profiles',
    metrics_tokens: 'Tokens per day (in/out/cache)',
    metrics_input: 'input',
    metrics_output: 'output',
    metrics_cache: 'cache',
    metrics_none: 'No data yet.',
    metrics_by_model: 'Tokens by model',
    metrics_download_json: 'Download JSON',
    metrics_download_csv: 'Download CSV',
    notify_unsupported: 'This browser does not support notifications.',
    notify_grant: 'Enable notifications',
    notify_denied: 'Notifications blocked in the browser.',
    notify_unmuted: 'Notifications enabled',
    notify_muted: 'Notifications muted',
    notify_action_login: 'Login',
    notify_action_session_new: 'New session',
    notify_action_session_kill: 'Session closed',
    notify_action_session_resume: 'Session resumed',
    notify_action_profile_apply: 'Profile applied',
    notify_action_turn_complete: 'Turn completed',
    notify_action_session_waiting: 'Waiting for approval',
    notify_action_session_choice: 'Decision needed',
    live_view: 'View session live',
    live_title: 'Session {0} live',
    live_closed: 'Session closed or disconnected.',
    live_reconnecting: 'Connection lost, reconnecting…',
    live_tab_chat: 'Chat',
    live_tab_term: 'Terminal',
    live_scroll_up: 'Scroll up',
    live_scroll_end: 'Go to end',
    live_screen_sep: '──── Current screen ────',
    mode_label: 'Mode',
    model_label: 'Model',
    chat_no_conv: 'No conversation yet (waiting for the first message…)',
    chat_placeholder: 'Type a message… (Enter sends, Shift+Enter newline)',
    chat_send: 'Send',
    chat_err: 'Could not send the message',
    chat_status_alive: 'Live',
    chat_status_off: 'Disconnected',
    chat_status_waiting: 'awaiting approval',
    chat_status_choice: 'choosing option',
    chat_approve: 'Approve',
    chat_approve_title: 'Approve the command (option 1)',
    chat_choice_title: 'Select option (Enter)',
    chat_choice_up: 'Previous option',
    chat_choice_down: 'Next option',
    chat_choice_hint: '↑/↓ to move, Enter or click to select',
    chat_stop_title: 'Cancel / Escape',
    mode_sent: 'Mode changed to {0}',
    model_sent: 'Model changed to {0}',
    chat_status_mode: 'mode: {0}',
    chat_status_model: 'model: {0}',
    chat_status_elapsed: '{0}',
    rc_reconnect_on: 'RC: re-register',
    rc_reconnect_off: 'RC: enable',
    rc_reconnect_title: 'Reconnect Remote Control: force a fresh bridge re-registration so the session appears in the Claude app.',
    confirm_rc_reconnect_on: 'It will re-register Remote Control; if the bridge cannot recover live, the session will be closed and relaunched (resuming the conversation). Continue?',
    confirm_rc_reconnect_off: 'It will enable Remote Control; if the bridge cannot recover live, the session will be closed and relaunched (resuming the conversation). Continue?',
    toast_rc_reconnect: 'Remote Control re-registered',
    toast_rc_recovered: 'Bridge not recoverable live: session relaunched as {0}',
    toast_rc_fail: 'Could not re-register Remote Control or relaunch the session ({0})',
  },
};

function ccsmApp() {
  return {
    // Auth state
    authenticated: false,
    lanBypass: false,
    userLabel: '',
    showLogin: true,
    loginUser: '',
    loginPass: '',
    loginError: '',

    // Data
    sessions: { loading: false, items: [], error: '' },
    profiles: [],
    conversations: { loading: false, items: [], page: 1, hasMore: false, error: '' },

    // UI state
    lang: 'es',
    viewMode: 'list',
    convSearch: '',
    convFilters: { origin: '', from: '', to: '', alive: false },
    actionLoading: false,
    // "New session" advanced form (optional tmux name, Claude name, profile,
    // project). project defaults to "principal" (home), the historical launch.
    adv: { tmux: '', claude: '', profile: '', project: 'principal' },
    projects: [],
    preview: { open: false, messages: [], date: '', origin: '', id: '', title: '', is_alive: false, tags: '', notes: '', saving: false },
    settings: { open: false, loading: false, groups: [], editing: null, editValue: '', users: [] },
    userModal: { open: false, mode: 'add', username: '', password: '', error: '' },
    audit: { open: false, loading: false, q: '', entries: [] },
    metrics: { open: false, loading: false, data: null, model: '' },
    notify: { supported: false, permission: 'default', muted: false, es: null },
    live: { open: false, name: '', view: 'chat', content: '', status: '', chatStatus: '', es: null, ces: null, timer: null, msgs: [], termHist: '', meta: null, input: '', sending: false, elapsed: '', models: [], maxH: null },
    rename: { open: false, session: '', tmuxName: '', claudeName: '' },
    profViewer: { open: false, name: '', html: '' },
    toast: { show: false, message: '', type: 'success' },
    toastTimer: null,

    // Polling
    pollInterval: null,

    // --- i18n ---
    initLang() {
      const saved = localStorage.getItem('ccsm_lang');
      this.lang = (saved === 'en' || saved === 'es') ? saved : 'es';
    },
    setLang(l) {
      this.lang = l;
      try { localStorage.setItem('ccsm_lang', l); } catch (e) { /* private mode */ }
    },
    t(key, vars) {
      const dict = I18N[this.lang] || I18N.es;
      let s = dict[key] !== undefined ? dict[key] : (I18N.es[key] !== undefined ? I18N.es[key] : key);
      if (vars) {
        vars.forEach((v, i) => { s = s.replace('{' + i + '}', String(v)); });
      }
      return s;
    },

    async init() {
      this.initLang();
      this.initViewportTrack();
      this.$watch('settings.open', v => this.setBodyLock());
      this.$watch('preview.open', v => this.setBodyLock());
      this.$watch('rename.open', v => this.setBodyLock());
      this.$watch('profViewer.open', v => this.setBodyLock());
      this.$watch('userModal.open', v => this.setBodyLock());
      await this.checkAuth();
      if (this.authenticated) {
        this.showLogin = false;
        this.loadAll();
        this.initNotify();
      }
    },

    setBodyLock() {
      document.body.style.overflow =
        (this.settings.open || this.preview.open || this.rename.open || this.profViewer.open || this.userModal.open) ? 'hidden' : '';
    },

    // Follows the visualViewport: when the mobile keyboard opens/closes, limit
    // the modal height (live.maxH) so it never leaves the visible area.
    initViewportTrack() {
      const setH = () => {
        const vv = window.visualViewport;
        const h = (vv && vv.height) ? vv.height : window.innerHeight;
        this.live.maxH = Math.round(h * 0.8);
      };
      setH();
      if (window.visualViewport) window.visualViewport.addEventListener('resize', setH);
      window.addEventListener('resize', setH);
    },

    // copyToClipboard tries navigator.clipboard, then falls back to execCommand
    // for non-secure (HTTP) contexts such as plain-LAN access.
    copyToClipboard(text) {
      try {
        navigator.clipboard.writeText(text).catch(() => { this.fallbackCopy(text); });
      } catch (e) {
        this.fallbackCopy(text);
      }
    },

    fallbackCopy(text) {
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.style.position = 'fixed';
      ta.style.left = '-9999px';
      ta.style.top = '-9999px';
      document.body.appendChild(ta);
      ta.focus();
      ta.select();
      try { document.execCommand('copy'); } catch (e) { /* last resort failed */ }
      document.body.removeChild(ta);
    },

    // --- Auth ---
    async checkAuth() {
      try {
        const resp = await fetch('/api/auth/status');
        const data = await resp.json();
        this.authenticated = data.authenticated;
        this.lanBypass = data.lan_bypass || false;
        this.userLabel = data.username || '';

        // If on LAN, auto-login
        if (!this.authenticated) {
          try {
            const resp2 = await fetch('/api/auth/login', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ username: '', password: '' }),
            });
            const data2 = await resp2.json();
            if (data2.lan_bypass) {
              this.authenticated = true;
              this.lanBypass = true;
              this.userLabel = this.t('lan_label');
            }
          } catch (e) { /* not on LAN */ }
        }
      } catch (e) {
        this.authenticated = false;
      }
    },

    async doLogin() {
      this.loginError = '';
      try {
        const resp = await fetch('/api/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username: this.loginUser, password: this.loginPass }),
        });
        const data = await resp.json();
        if (data.ok) {
          this.authenticated = true;
          this.showLogin = false;
          this.userLabel = this.loginUser;
          this.loginPass = '';
          this.loadAll();
          this.initNotify();
        } else {
          this.loginError = this.t('toast_error_auth');
        }
      } catch (e) {
        this.loginError = this.t('toast_error_conn_login');
      }
    },

    async logout() {
      await fetch('/api/auth/logout', { method: 'POST' });
      this.authenticated = false;
      this.showLogin = true;
      this.userLabel = '';
      if (this.pollInterval) clearInterval(this.pollInterval);
      this.stopNotify();
    },

    // --- Data loading ---
    async loadAll() {
      await Promise.all([
        this.loadSessions(),
        this.loadConversations(),
        this.loadProfiles(),
        this.loadProjects(),
      ]);
      if (this.pollInterval) clearInterval(this.pollInterval);
      this.pollInterval = setInterval(() => this.loadSessions(), 30000);
    },

    async reloadAll() {
      await this.loadAll();
      this.toastMsg(this.t('refresh'), 'success');
    },

    async loadSessions() {
      this.sessions.loading = true;
      this.sessions.error = '';
      try {
        const resp = await fetch('/api/sessions');
        if (!resp.ok) throw new Error(await resp.text());
        this.sessions.items = await resp.json();
      } catch (e) {
        this.sessions.error = e.message;
      }
      this.sessions.loading = false;
    },

    async loadConversations(page = 1) {
      this.conversations.loading = true;
      this.conversations.error = '';
      try {
        const params = new URLSearchParams({ page: String(page), per_page: '20' });
        if (this.convSearch) params.set('q', this.convSearch);
        if (this.convFilters.origin) params.set('origin', this.convFilters.origin);
        if (this.convFilters.from) params.set('from', this.convFilterDate(this.convFilters.from));
        if (this.convFilters.to) params.set('to', this.convFilterDate(this.convFilters.to));
        if (this.convFilters.alive) params.set('alive', '1');
        const resp = await fetch('/api/conversations?' + params);
        if (!resp.ok) throw new Error(await resp.text());
        const items = await resp.json();
        if (page === 1) {
          this.conversations.items = items;
        } else {
          this.conversations.items = [...this.conversations.items, ...items];
        }
        this.conversations.page = page;
        this.conversations.hasMore = items.length >= 20;
      } catch (e) {
        this.conversations.error = e.message;
      }
      this.conversations.loading = false;
    },

    applyConvFilters() {
      this.conversations.items = [];
      this.loadConversations(1);
    },

    // HTML <input type=date> gives YYYY-MM-DD; the API uses DD/MM/YYYY.
    convFilterDate(iso) {
      if (!iso) return '';
      const p = iso.split('-');
      if (p.length !== 3) return iso;
      return p[2] + '/' + p[1] + '/' + p[0];
    },

    async loadMoreConversations() {
      await this.loadConversations(this.conversations.page + 1);
    },

    async loadProfiles() {
      try {
        const resp = await fetch('/api/profiles');
        if (resp.ok) this.profiles = await resp.json();
      } catch (e) { /* ignore */ }
    },

    async loadProjects() {
      try {
        const resp = await fetch('/api/projects');
        if (resp.ok) this.projects = await resp.json();
      } catch (e) { /* ignore */ }
    },

    // Short label for a project: its name relative to home already is unique;
    // the dropdown shows just the base dir, and the session badge too.
    projectLabel(rel) {
      const p = String(rel || '').split('/');
      return p[p.length - 1] || rel;
    },

    // Projects for the dropdown, sorted by the visible label (base dir), with
    // "principal" handled as a fixed first option in the markup.
    sortedProjects() {
      return this.projects
        .filter(p => p.name !== 'principal')
        .sort((a, b) => {
          const la = this.projectLabel(a.name);
          const lb = this.projectLabel(b.name);
          return la === lb ? a.name.localeCompare(b.name) : la.localeCompare(lb);
        });
    },

    // Relative path (from the dir Claude runs in) of the selected project, for
    // the hint under the dropdown. Home is the launch dir itself, shown as "~".
    selectedProjectPath() {
      const p = this.adv.project;
      if (!p || p === 'principal') return '~';
      return p;
    },

    // --- Live session view (SSE): Terminal + Chat ---
    openLive(s) {
      this.closeLive();
      this.live.open = true;
      this.live.name = s.name;
      this.live.content = '';
      this.live.status = '';
      this.live.chatStatus = '';
      this.live.msgs = [];
      this.live.termHist = '';
      this.live.meta = null;
      this.live.input = '';
      this.live.elapsed = '';
      this.live.view = 'chat';
      this.loadModels();
      this.startChatStream();
    },

    // Available models: opus/sonnet/haiku + whatever appears in the applied
    // settings.json (e.g. ANTHROPIC_DEFAULT_*_MODEL).
    async loadModels() {
      const base = ['opus', 'sonnet', 'haiku'];
      let extra = [];
      try {
        const resp = await fetch('/api/settings');
        if (resp.ok) {
          const d = await resp.json();
          let cfg = {};
          try { cfg = typeof d.content === 'string' ? JSON.parse(d.content) : (d.content || {}); } catch (e) {}
          if (cfg.model) extra.push(String(cfg.model));
          const env = cfg.env || {};
          const keys = Object.keys(env).filter(k => /^ANTHROPIC_(DEFAULT_(OPUS|SONNET|HAIKU)_MODEL|SMALL_FAST_MODEL|MODEL)$|^CLAUDE_CODE_(SUBAGENT|BG_CLASSIFIER)_MODEL$/.test(k));
          keys.forEach(k => { if (env[k]) extra.push(String(env[k])); });
        }
      } catch (e) { /* no extra models */ }
      const seen = new Set();
      this.live.models = base.concat(extra).filter(m => {
        m = m.trim();
        if (!m || seen.has(m)) return false;
        seen.add(m);
        return true;
      });
    },

    // The mode dropdown. Once the host has calibrated the session's real
    // Shift+Tab wheel (auto/bypassPermissions only appear when the account
    // enables them), the chat payload carries it in meta.modes; before that we
    // offer the standard four. The first switch calibrates and the list adapts.
    modeOptions() {
      const w = this.live.meta && this.live.meta.modes;
      if (w && w.length) return w;
      return ['auto', 'plan', 'accept-edits', 'manual'];
    },

    // Changes the session mode. /mode does not exist in Claude Code (2.1.227):
    // the host resolves it with /plan + the Shift+Tab wheel, so we send the
    // mode as a structured {mode} request, not as chat text.
    async setMode(mode) {
      if (!mode) return;
      try {
        const resp = await fetch('/api/sessions/' + encodeURIComponent(this.live.name) + '/send', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ mode }),
        });
        if (resp.ok) {
          this.toastSuccess(this.t('mode_sent', [mode]));
        } else {
          const b = await resp.json().catch(() => ({}));
          this.toastError((b && b.error) || this.t('chat_err'));
        }
      } catch (e) {
        this.toastError(this.t('chat_err'));
      }
    },

    // Changes the session model by typing /model <x> into the pane.
    async setModel(model) {
      if (!model) return;
      if (await this.sendChatText('/model ' + model)) {
        this.toastSuccess(this.t('model_sent', [model]));
      }
    },

    setLiveView(v) {
      if (v === this.live.view) return;
      if (v === 'term') {
        this.closeChatStream();
        this.startTermStream();
      } else {
        this.closeTermStream();
        this.startChatStream();
      }
      this.live.view = v;
    },

    atBottom(el) {
      return el.scrollHeight - el.scrollTop - el.clientHeight < 8;
    },

    // dir -1: scroll up one page; dir 1: go to the end. Same behaviour in the
    // chat and the terminal views.
    scrollLive(dir) {
      const el = this.live.view === 'chat' ? this.$refs.liveChat : this.$refs.livePane;
      if (!el) return;
      if (dir > 0) {
        el.scrollTop = el.scrollHeight;
      } else {
        el.scrollTop = Math.max(0, el.scrollTop - el.clientHeight);
      }
    },

    // Terminal tab: conversation history + current screen. Claude
    // Code paints in tmux's alternate screen (no scrollback), so the terminal
    // history comes from the transcript (same as the chat).
    get termText() {
      const h = this.live.termHist || '';
      const s = this.live.content || '';
      if (h && s) return h + '\n\n' + this.t('live_screen_sep') + '\n' + s;
      if (h) return h;
      return s;
    },

    startTermStream() {
      this.live.content = '';
      this.live.status = '';
      const es = new EventSource('/api/sessions/' + encodeURIComponent(this.live.name) + '/stream');
      this.live.es = es;
      es.onopen = () => { this.live.status = ''; };
      es.onmessage = (ev) => {
        const el = this.$refs.livePane;
        const stick = el ? this.atBottom(el) : true;
        this.live.content = ev.data.replace(/\\n/g, '\n');
        this.live.status = '';
        this.$nextTick(() => {
          if (stick && el) el.scrollTop = el.scrollHeight;
        });
      };
      // Don't close() on error: EventSource reconnects on its own (browser
      // retry), and closing it here would kill that automatic reconnect.
      // onopen clears the "closed" status once the stream comes back.
      es.onerror = () => {
        this.live.status = this.t('live_reconnecting');
      };
    },

    closeTermStream() {
      if (this.live.es) { this.live.es.close(); this.live.es = null; }
    },

    startChatStream() {
      this.live.msgs = [];
      this.live.termHist = '';
      this.live.meta = null;
      this.live.chatStatus = '';
      this.loadChat();
      this.live.timer = setInterval(() => {
        this.live.elapsed = this.fmtElapsed();
      }, 1000);
      const es = new EventSource('/api/sessions/' + encodeURIComponent(this.live.name) + '/chat/stream');
      this.live.ces = es;
      es.onopen = () => { this.live.chatStatus = ''; };
      es.onmessage = (ev) => {
        this.applyChat(ev.data);
        this.live.chatStatus = '';
      };
      // Don't close() on error: EventSource reconnects on its own (browser
      // retry), and closing it here would kill that automatic reconnect.
      // onopen clears the "closed" status once the stream comes back.
      es.onerror = () => {
        this.live.chatStatus = this.t('live_reconnecting');
      };
    },

    closeChatStream() {
      if (this.live.ces) { this.live.ces.close(); this.live.ces = null; }
      if (this.live.timer) { clearInterval(this.live.timer); this.live.timer = null; }
    },

    async loadChat() {
      try {
        const resp = await fetch('/api/sessions/' + encodeURIComponent(this.live.name) + '/chat');
        if (!resp.ok) { this.live.chatStatus = this.t('live_closed'); return; }
        this.applyChat(await resp.text());
      } catch (e) {
        this.live.chatStatus = this.t('live_closed');
      }
    },

    applyChat(raw) {
      let data;
      try { data = JSON.parse(raw); } catch (e) { return; }
      const el = this.$refs.liveChat;
      const stick = el ? this.atBottom(el) : true;
      this.live.meta = data;
      // No leading/trailing blank lines in each message.
      const msgs = (data.messages || []).map(m => Object.assign({}, m, {
        content: String(m.content || '').trim(),
      }));
      this.live.msgs = msgs;
      // Terminal history: the conversation rendered as terminal output.
      this.live.termHist = msgs.map(m => (m.role === 'user' ? '❯ ' : '') + m.content).join('\n\n');
      this.$nextTick(() => {
        if (this.live.view === 'term') {
          const tp = this.$refs.livePane;
          if (tp && this.atBottom(tp)) tp.scrollTop = tp.scrollHeight;
        } else if (stick && el) {
          el.scrollTop = el.scrollHeight;
        }
      });
    },

    fmtElapsed() {
      const created = this.live.meta && this.live.meta.created;
      if (!created) return '';
      const s = Math.max(0, Math.floor(Date.now() / 1000 - created));
      const m = Math.floor(s / 60);
      const h = Math.floor(m / 60);
      if (h > 0) return h + 'h ' + (m % 60) + 'm';
      if (m > 0) return m + 'm ' + (s % 60) + 's';
      return s + 's';
    },

    // Enter sends; Shift+Enter inserts a newline. The .exact modifier is not in
    // the Alpine bundle (3.14.9): it blocks the whole listener, and Enter ended
    // up inserting a newline without sending. So the decision is made here with
    // shiftKey instead of relying on .exact.
    onChatKeydown(e) {
      if (e.shiftKey) return;   // Shift+Enter: default behaviour (newline)
      e.preventDefault();       // Enter: stop the newline and send
      this.sendChat();
    },

    async sendChat() {
      const text = this.live.input;
      if (!text.trim()) return;
      this.live.input = '';
      this.live.sending = true;
      // Refresh right away instead of waiting for the next 1s SSE poll tick,
      // so the sent message appears immediately.
      if (await this.sendChatText(text)) this.loadChat();
      this.live.sending = false;
    },

    async sendChatText(text) {
      if (!text || !text.trim()) return false;
      try {
        const resp = await fetch('/api/sessions/' + encodeURIComponent(this.live.name) + '/send', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ text: text.trim() }),
        });
        if (!resp.ok) {
          const b = await resp.json().catch(() => ({}));
          this.toastError((b && b.error) || this.t('chat_err'));
          return false;
        }
        return true;
      } catch (e) {
        this.toastError(this.t('chat_err'));
        return false;
      }
    },

    async sendKey(key) {
      try {
        const resp = await fetch('/api/sessions/' + encodeURIComponent(this.live.name) + '/send', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ keys: key }),
        });
        if (!resp.ok) {
          const b = await resp.json().catch(() => ({}));
          this.toastError((b && b.error) || this.t('chat_err'));
        }
      } catch (e) {
        this.toastError(this.t('chat_err'));
      }
    },

    closeLive() {
      this.closeTermStream();
      this.closeChatStream();
      this.live.open = false;
    },

    // --- Settings panel ---
    async openSettings() {
      this.settings.open = true;
      if (this.settings.groups.length) return;
      this.settings.loading = true;
      try {
        const resp = await fetch('/api/config');
        if (!resp.ok) throw new Error('config unavailable');
        const cfg = await resp.json();
        this.settings.groups = this.buildConfigGroups(cfg);
      } catch (e) {
        this.toastMsg(e.message, 'error');
      }
      this.settings.loading = false;
    },

    // --- Audit log ---
    openAudit() {
      this.audit.open = true;
      this.loadAudit();
    },

    openMetrics() {
      this.metrics.open = true;
      this.loadMetrics();
    },

    async loadMetrics() {
      this.metrics.loading = true;
      try {
        const q = this.metrics.model ? '?model=' + encodeURIComponent(this.metrics.model) : '';
        const resp = await fetch('/api/metrics' + q);
        if (!resp.ok) throw new Error('metrics unavailable');
        this.metrics.data = await resp.json();
      } catch (e) {
        this.metrics.data = null;
      }
      this.metrics.loading = false;
    },

    downloadMetrics(format) {
      const d = this.metrics.data;
      if (!d) return;
      const stamp = new Date().toISOString().slice(0, 10);
      if (format === 'json') {
        this.downloadBlob(JSON.stringify(d, null, 2), 'application/json', `ccsm-metrics-${stamp}.json`);
        return;
      }
      // CSV: per-day rows; add per-model rows when not filtered by a single model.
      const rows = [];
      (d.token_usage_per_day || []).forEach(r => rows.push(['tokens', r.date, r.input, r.output, r.cache]));
      (d.sessions_per_day || []).forEach(r => rows.push(['sessions', r.date, r.count, '', '']));
      (d.token_usage_by_model || []).forEach(r => rows.push(['model:' + r.model, r.messages + ' msg', r.input, r.output, r.cache]));
      const esc = v => String(v).replace(/"/g, '""');
      const csv = ['type,date,input,output,cache']
        .concat(rows.map(r => r.map(v => `"${esc(v)}"`).join(',')))
        .join('\n');
      this.downloadBlob(csv, 'text/csv', `ccsm-metrics-${stamp}.csv`);
    },

    downloadBlob(content, mime, filename) {
      const blob = new Blob([content], { type: mime });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    },

    fmtTokens(n) {
      if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M';
      if (n >= 1e3) return (n / 1e3).toFixed(0) + 'k';
      return String(n);
    },

    fmtUptime(s) {
      const d = Math.floor(s / 86400);
      const h = Math.floor((s % 86400) / 3600);
      const m = Math.floor((s % 3600) / 60);
      const parts = [];
      if (d) parts.push(d + 'd');
      if (h) parts.push(h + 'h');
      if (m) parts.push(m + 'm');
      return parts.join(' ') || '0m';
    },

    maxBar(items, key) {
      let max = 0;
      for (const it of items || []) {
        const v = it[key] || 0;
        if (v > max) max = v;
      }
      return max;
    },

    // --- Notifications (web push via SSE + Notification API) ---
    initNotify() {
      if (this.notify.es) return; // already connected (re-login without reload)
      this.notify.supported = 'Notification' in window;
      if (!this.notify.supported) return;
      this.notify.permission = Notification.permission;
      this.notify.muted = localStorage.getItem('ccsm_notify_muted') === '1';
      try {
        const es = new EventSource('/api/events');
        this.notify.es = es;
        es.onmessage = (e) => {
          let ev;
          try { ev = JSON.parse(e.data); } catch (err) { return; }
          if (document.hidden && this.notify.permission === 'granted' && !this.notify.muted) {
            new Notification('CCSM · ' + this.notifyTitle(ev.action), {
              body: ev.detail || ev.user || ev.action,
              tag: 'ccsm-event'
            });
          }
        };
        es.onerror = () => { /* EventSource reconnects on its own */ };
      } catch (e) { /* noop */ }
    },

    stopNotify() {
      if (this.notify.es) {
        this.notify.es.close();
        this.notify.es = null;
      }
    },

    async toggleNotify() {
      if (!this.notify.supported) {
        this.toastMsg(this.t('notify_unsupported'), 'error');
        return;
      }
      if (this.notify.permission === 'granted') {
        this.notify.muted = !this.notify.muted;
        localStorage.setItem('ccsm_notify_muted', this.notify.muted ? '1' : '0');
        this.toastMsg(this.t(this.notify.muted ? 'notify_muted' : 'notify_unmuted'), 'success');
        return;
      }
      if (this.notify.permission === 'denied') {
        this.toastMsg(this.t('notify_denied'), 'error');
        return;
      }
      const perm = await Notification.requestPermission();
      this.notify.permission = perm;
      if (perm === 'granted') {
        this.toastMsg(this.t('notify_unmuted'), 'success');
      } else {
        this.toastMsg(this.t('notify_denied'), 'error');
      }
    },

    notifyTitle(action) {
      const key = 'notify_action_' + action;
      return (key in I18N[this.lang]) ? this.t(key) : action;
    },

    async loadAudit() {
      this.audit.loading = true;
      try {
        const resp = await fetch('/api/audit?n=200');
        if (!resp.ok) throw new Error('audit unavailable');
        const data = await resp.json();
        this.audit.entries = data.entries || [];
      } catch (e) {
        this.audit.entries = [];
        this.toastMsg(e.message, 'error');
      }
      this.audit.loading = false;
    },

    filteredAudit() {
      const q = this.audit.q.trim().toLowerCase();
      if (!q) return this.audit.entries;
      return this.audit.entries.filter((e) =>
        (e.action || '').toLowerCase().includes(q) ||
        (e.user || '').toLowerCase().includes(q) ||
        (e.detail || '').toLowerCase().includes(q));
    },

    auditTime(iso) {
      const d = new Date(iso);
      if (isNaN(d)) return iso;
      return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    },

    auditBadge(action) {
      if (action === 'login_failed') return 'bg-danger/15 text-danger';
      if (action.startsWith('session_')) return 'bg-accent/15 text-accent';
      if (action.startsWith('user_') || action === 'config_update') return 'bg-warn/15 text-warn';
      return 'bg-success/15 text-success';
    },

    buildConfigGroups(cfg) {
      const t = (k) => this.t(k);
      this.settings.users = cfg.users || [];
      return [
        {
          title: t('cfg_deploy'),
          rows: [
            { k: t('cfg_mode'), v: cfg.mode, editable: false },
            { k: t('cfg_port'), v: String(cfg.port), editable: false },
            { k: t('cfg_attach'), v: cfg.host_attach_addr || '—', editable: true, field: 'host_attach_addr', type: 'text', raw: cfg.host_attach_addr },
          ],
        },
        {
          title: t('cfg_paths'),
          rows: [
            { k: 'conversations', v: cfg.paths.conversations, editable: false },
            { k: 'profiles', v: cfg.paths.profiles, editable: false },
            { k: 'settings', v: cfg.paths.settings, editable: false },
            { k: 'claude_binary', v: cfg.paths.claude_binary, editable: false },
            { k: 'tmux_binary', v: cfg.paths.tmux_binary, editable: false },
            { k: 'bash_binary', v: cfg.paths.bash_binary, editable: false },
          ],
        },
        {
          title: t('cfg_security'),
          rows: [
            { k: t('cfg_lan'), v: (cfg.lan_subnets || []).join(', ') || '—', editable: true, field: 'lan_subnets', type: 'csv', raw: cfg.lan_subnets },
            { k: t('cfg_socket'), v: cfg.agent_socket || t('cfg_direct'), editable: false },
          ],
        },
        {
          title: t('cfg_rc'),
          rows: [
            { k: t('cfg_rc_bootstrap'), v: cfg.rc.bootstrap_profile, editable: true, field: 'rc.bootstrap_profile', type: 'text', raw: cfg.rc.bootstrap_profile },
            { k: t('cfg_rc_wait'), v: String(cfg.rc.wait_seconds) + 's', editable: true, field: 'rc.wait_seconds', type: 'number', raw: cfg.rc.wait_seconds },
            { k: t('cfg_rc_poll'), v: String(cfg.rc.poll_seconds) + 's', editable: true, field: 'rc.poll_seconds', type: 'number', raw: cfg.rc.poll_seconds },
          ],
        },
      ];
    },


    startEdit(row) {
      this.settings.editing = row.field;
      if (row.type === 'csv') {
        this.settings.editValue = (row.raw || []).join(', ');
      } else if (row.raw !== undefined && row.raw !== null) {
        this.settings.editValue = String(row.raw);
      } else {
        this.settings.editValue = row.v;
      }
    },
    cancelEdit() {
      this.settings.editing = null;
      this.settings.editValue = '';
    },
    async saveEdit(row) {
      const body = {};
      if (row.field === 'lan_subnets') {
        body.lan_subnets = this.settings.editValue.split(',').map(s => s.trim()).filter(s => s);
      } else if (row.field === 'host_attach_addr') {
        body.host_attach_addr = this.settings.editValue.trim();
      } else if (row.field.startsWith('rc.')) {
        body.rc = {};
        const key = row.field.replace('rc.', '');
        if (row.type === 'number') {
          const n = parseInt(this.settings.editValue, 10);
          if (!Number.isFinite(n)) {
            this.toastMsg(this.t('name_invalid'), 'error');
            return;
          }
          body.rc[key] = n;
        } else {
          body.rc[key] = this.settings.editValue.trim();
        }
      }
      try {
        const resp = await fetch('/api/config', {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        });
        const data = await resp.json();
        if (data.ok) {
          const msg = data.restart_needed ? this.t('cfg_saved') + ' ' + this.t('cfg_restart_needed') : this.t('cfg_saved');
          this.toastMsg(msg, 'success');
          this.settings.editing = null;
          // Reload config
          this.settings.loading = true;
          try {
            const resp2 = await fetch('/api/config');
            if (resp2.ok) {
              const cfg2 = await resp2.json();
              this.settings.groups = this.buildConfigGroups(cfg2);
            }
          } catch(e) {}
          this.settings.loading = false;
        } else {
          this.toastMsg(data.error || 'Error', 'error');
        }
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
      }
    },
    // User management
    openAddUser() {
      this.userModal.mode = 'add';
      this.userModal.username = '';
      this.userModal.password = '';
      this.userModal.error = '';
      this.userModal.open = true;
    },
    openChangePassword(username) {
      this.userModal.mode = 'password';
      this.userModal.username = username;
      this.userModal.password = '';
      this.userModal.error = '';
      this.userModal.open = true;
    },
    async doAddUser() {
      this.userModal.error = '';
      if (!this.userModal.username) {
        this.userModal.error = this.t('username') + ' required';
        return;
      }
      if (!this.userModal.password || this.userModal.password.length < 8) {
        this.userModal.error = this.t('cfg_password_label');
        return;
      }
      try {
        const resp = await fetch('/api/config/users', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username: this.userModal.username, password: this.userModal.password }),
        });
        const data = await resp.json();
        if (data.ok) {
          this.toastMsg(this.t('cfg_user_added', [this.userModal.username]), 'success');
          this.userModal.open = false;
          // Reload users
          try {
            const r = await fetch('/api/config/users');
            if (r.ok) this.settings.users = await r.json();
          } catch(e) {}
        } else {
          this.userModal.error = data.error || 'Error';
        }
      } catch (e) {
        this.userModal.error = e.message;
      }
    },
    async doChangePassword() {
      this.userModal.error = '';
      if (!this.userModal.password || this.userModal.password.length < 8) {
        this.userModal.error = this.t('cfg_password_label');
        return;
      }
      try {
        const resp = await fetch('/api/config/users/' + encodeURIComponent(this.userModal.username) + '/password', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ password: this.userModal.password }),
        });
        const data = await resp.json();
        if (data.ok) {
          this.toastMsg(this.t('cfg_password_changed'), 'success');
          this.userModal.open = false;
        } else {
          this.userModal.error = data.error || 'Error';
        }
      } catch (e) {
        this.userModal.error = e.message;
      }
    },
    async deleteUser(username) {
      if (!confirm(this.t('cfg_confirm_del_user', [username]))) return;
      try {
        const resp = await fetch('/api/config/users/' + encodeURIComponent(username), { method: 'DELETE' });
        const data = await resp.json();
        if (data.ok) {
          this.toastMsg(this.t('cfg_user_deleted', [username]), 'success');
          try {
            const r = await fetch('/api/config/users');
            if (r.ok) this.settings.users = await r.json();
          } catch(e) {}
        } else {
          this.toastMsg(data.error || 'Error', 'error');
        }
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
      }
    },

    async copyText(text) {
      this.copyToClipboard(text);
      this.toastMsg(this.t('cfg_copy') + ': ' + text, 'success');
    },

    // --- Actions ---
    async createSession(opts = {}) {
      this.actionLoading = true;
      try {
        const payload = {};
        if (opts.profile) payload.profile = opts.profile;
        if (opts.name) payload.name = opts.name;
        if (opts.claudeName) payload.claude_name = opts.claudeName;
        if (opts.project && opts.project !== 'principal') payload.project = opts.project;
        const resp = await fetch('/api/sessions/new', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
        const data = await resp.json();
        if (data.ok) {
          this.toastMsg(this.t('toast_session_created', [data.session_name]), 'success');
          await this.loadSessions();
          return data;
        }
        this.toastMsg(data.error || this.t('toast_error_create'), 'error');
        return null;
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
        return null;
      } finally {
        this.actionLoading = false;
      }
    },

    // Quick session: everything default, then jump straight into its chat.
    async newSessionQuick() {
      const data = await this.createSession({});
      if (data && data.session_name) {
        this.openLive({ name: data.session_name });
      }
    },

    // Advanced session: optional tmux name, Claude name and profile. Like the
    // quick button, it jumps straight into the new session's chat.
    async newSessionAdvanced() {
      const tmux = this.adv.tmux.trim();
      const claude = this.adv.claude.trim();
      if (tmux && !/^[A-Za-z0-9_-]{1,32}$/.test(tmux)) {
        this.toastMsg(this.t('name_invalid'), 'error');
        return;
      }
      if (claude && !/^[\p{L}\p{N}\p{P} ]{1,80}$/u.test(claude)) {
        this.toastMsg(this.t('name_invalid'), 'error');
        return;
      }
      const data = await this.createSession({ name: tmux, claudeName: claude, profile: this.adv.profile, project: this.adv.project });
      this.adv = { tmux: '', claude: '', profile: '', project: 'principal' };
      if (data && data.session_name) {
        this.openLive({ name: data.session_name });
      }
    },

    async resumeSession(id) {
      this.actionLoading = true;
      try {
        const resp = await fetch('/api/sessions/resume', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ id }),
        });
        const data = await resp.json();
        if (data.ok) {
          this.toastMsg(this.t('toast_session_resumed', [data.session_name]), 'success');
          if (data.attach_cmd) {
            try { await navigator.clipboard.writeText(data.attach_cmd); } catch (e) { /* ok */ }
          }
          await this.loadSessions();
        } else {
          this.toastMsg(data.error || this.t('toast_error_resume'), 'error');
        }
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
      }
      this.actionLoading = false;
    },

    async killSession(s) {
      if (!confirm(this.t('confirm_kill', [s.name]))) return;
      try {
        const resp = await fetch('/api/sessions/' + encodeURIComponent(s.name), { method: 'DELETE' });
        if (resp.ok) {
          this.toastMsg(this.t('toast_session_killed', [s.name]), 'success');
          await this.loadSessions();
        }
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
      }
    },

    async reconnectRC() {
      const status = this.live.meta && this.live.meta.status;
      const msg = status === 'rc_connected'
        ? this.t('confirm_rc_reconnect_on')
        : this.t('confirm_rc_reconnect_off');
      if (!confirm(msg)) return;
      try {
        const resp = await fetch('/api/sessions/' + encodeURIComponent(this.live.name) + '/rc', { method: 'POST' });
        const data = await resp.json().catch(() => ({}));
        if (!resp.ok) {
          this.toastMsg(data.error || this.t('chat_err'), 'error');
          return;
        }
        if (data.recovered) {
          this.toastMsg(this.t('toast_rc_recovered', [data.session]), 'success');
          this.openLive({ name: data.session });
          return;
        }
        if (data.status && data.status !== 'ok' && data.status !== 'rc_connected') {
          this.toastMsg(this.t('toast_rc_fail', [data.status]), 'error');
          return;
        }
        this.toastMsg(this.t('toast_rc_reconnect'), 'success');
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
      }
    },

    async applyProfile(name) {
      this.actionLoading = true;
      try {
        const resp = await fetch('/api/profiles/apply', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ profile: name }),
        });
        if (resp.ok) {
          this.toastMsg(this.t('toast_profile_applied', [name]), 'success');
          await this.loadProfiles();
        } else {
          const data = await resp.json();
          this.toastMsg(data.error || this.t('toast_error_auth'), 'error');
        }
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
      }
      this.actionLoading = false;
    },

    async copyAttach(s) {
      const cmd = s.attach_cmd || 'ssh ' + (s.host || 'host') + ' -t tmux a -t ' + s.name;
      this.copyToClipboard(cmd);
      this.toastMsg(this.t('toast_attach_copied', [cmd]), 'success');
    },

    async previewConversation(c) {
      this.preview.id = c.id;
      this.preview.date = c.date || '';
      this.preview.origin = c.origin || '';
      this.preview.title = c.title || '';
      this.preview.is_alive = !!c.is_alive;
      this.preview.tags = (c.tags || []).join(', ');
      this.preview.notes = c.notes || '';
      try {
        const resp = await fetch('/api/conversations/' + c.id + '?lines=20');
        if (resp.ok) {
          const data = await resp.json();
          this.preview.messages = data.messages || [];
          if (data.title) this.preview.title = data.title;
          if (typeof data.is_alive === 'boolean') this.preview.is_alive = data.is_alive;
        }
      } catch (e) {
        this.preview.messages = [];
      }
      try {
        const meta = await fetch('/api/conversations/' + c.id + '/meta');
        if (meta.ok) {
          const m = await meta.json();
          this.preview.tags = (m.tags || []).join(', ');
          this.preview.notes = m.notes || '';
        }
      } catch (e) { /* keep list values */ }
      this.preview.open = true;
    },

    async savePreviewMeta() {
      const id = this.preview.id;
      if (!id) return;
      this.preview.saving = true;
      try {
        const tags = this.preview.tags.split(',').map(s => s.trim()).filter(Boolean);
        const resp = await fetch('/api/conversations/' + id + '/meta', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ tags, notes: this.preview.notes, pinned: this.pinned(id), archived: this.archived(id) })
        });
        if (!resp.ok) {
          const err = await resp.json();
          this.toastMsg(err.error || this.t('toast_error_meta'), 'error');
          return;
        }
        this.toastMsg(this.t('meta_saved'), 'success');
        this.applyConvFilters();
      } catch (e) {
        this.toastMsg(this.t('toast_error_meta'), 'error');
      } finally {
        this.preview.saving = false;
      }
    },

    async togglePinned(c) {
      c.pinned = !c.pinned;
      await this.setConvMeta(c, { pinned: c.pinned });
    },

    async toggleArchived(c) {
      c.archived = !c.archived;
      await this.setConvMeta(c, { archived: c.archived });
    },

    async setConvMeta(c, patch) {
      try {
        const body = { tags: c.tags || [], notes: c.notes || '', pinned: !!c.pinned, archived: !!c.archived, ...patch };
        const resp = await fetch('/api/conversations/' + c.id + '/meta', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body)
        });
        if (!resp.ok) throw new Error(await resp.text());
      } catch (e) {
        if ('pinned' in patch) c.pinned = !c.pinned;
        if ('archived' in patch) c.archived = !c.archived;
        this.toastMsg(this.t('toast_error_meta'), 'error');
        return;
      }
      if ('pinned' in patch) {
        this.applyConvFilters();
      } else if (patch.archived) {
        this.conversations.items = this.conversations.items.filter(x => x.id !== c.id);
      }
    },

    pinned(id) {
      const c = this.conversations.items.find(x => x.id === id);
      return !!(c && c.pinned);
    },

    archived(id) {
      const c = this.conversations.items.find(x => x.id === id);
      return !!(c && c.archived);
    },

    // --- Profile viewer ---
    async viewProfile(name) {
      this.profViewer.name = name;
      this.profViewer.html = '<span class="text-fg-muted">' + escapeHtml(this.t('loading')) + '</span>';
      this.profViewer.open = true;
      try {
        const resp = await fetch('/api/profiles/' + name);
        if (!resp.ok) {
          this.profViewer.html = '<span class="text-danger">Error: ' + escapeHtml(await resp.text()) + '</span>';
          return;
        }
        const data = await resp.json();
        this.profViewer.html = highlightJSON(data.content);
      } catch (e) {
        this.profViewer.html = '<span class="text-danger">' + escapeHtml(e.message) + '</span>';
      }
    },

    // --- Rename ---
    openRename(s) {
      this.rename.session = s.name;
      this.rename.tmuxName = s.name;
      this.rename.claudeName = s.task || '';
      this.rename.open = true;
    },

    async renameTmux() {
      const newName = this.rename.tmuxName.trim();
      if (!newName || !/^[A-Za-z0-9_-]{1,32}$/.test(newName)) {
        this.toastMsg(this.t('name_invalid'), 'error');
        return;
      }
      try {
        const resp = await fetch('/api/sessions/' + encodeURIComponent(this.rename.session) + '/rename', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ new_name: newName }),
        });
        if (resp.ok) {
          this.toastMsg(this.t('toast_session_renamed', [newName]), 'success');
          this.rename.session = newName;
          this.rename.tmuxName = newName;
          await this.loadSessions();
        } else {
          const data = await resp.json();
          this.toastMsg(data.error || 'Error', 'error');
        }
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
      }
    },

    async renameClaude() {
      const title = this.rename.claudeName.trim();
      if (!title || title.length > 80) {
        this.toastMsg(this.t('name_invalid'), 'error');
        return;
      }
      try {
        const resp = await fetch('/api/sessions/' + encodeURIComponent(this.rename.session) + '/claude-name', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ title: title }),
        });
        if (resp.ok) {
          this.toastMsg(this.t('toast_claude_renamed'), 'success');
          await this.loadSessions();
        } else {
          const data = await resp.json();
          this.toastMsg(data.error || 'Error', 'error');
        }
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
      }
    },

    async viewCurrentSettings() {
      this.profViewer.name = this.t('cfg_viewing_settings');
      this.profViewer.html = '<span class="text-fg-muted">' + escapeHtml(this.t('loading')) + '</span>';
      this.profViewer.open = true;
      try {
        const resp = await fetch('/api/settings');
        if (!resp.ok) {
          this.profViewer.html = '<span class="text-danger">Error: ' + escapeHtml(await resp.text()) + '</span>';
          return;
        }
        const data = await resp.json();
        this.profViewer.html = highlightJSON(data.content);
      } catch (e) {
        this.profViewer.html = '<span class="text-danger">' + escapeHtml(e.message) + '</span>';
      }
    },

    // --- Toast ---
    toastMsg(msg, type) {
      if (this.toastTimer) clearTimeout(this.toastTimer);
      this.toast.show = true;
      this.toast.message = msg;
      this.toast.type = type;
      this.toastTimer = setTimeout(() => { this.toast.show = false; }, 4000);
    },
    toastError(msg) { this.toastMsg(msg, 'error'); },
    toastSuccess(msg) { this.toastMsg(msg, 'success'); },
  };
}

// escapeHtml neutralizes a string for safe interpolation into x-html content.
function escapeHtml(raw) {
  return String(raw).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// highlightJSON returns syntax-highlighted HTML for a JSON string.
// Token types: keys=cyan, strings=green, numbers=yellow, booleans/null=magenta,
// brackets/braces=dimmed, commas=white.
function highlightJSON(raw) {
  const escaped = escapeHtml(raw);
  return escaped.replace(
    /("(?:\\.|[^"\\])*")\s*:|("(?:\\.|[^"\\])*")|(-?\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b)|(\b(?:true|false|null)\b)|([{}[\]])|(,)/g,
    function(m, key, str, num, word, brack, comma) {
      if (key) return '<span class="json-key">' + key + '</span>:';
      if (str) return '<span class="json-string">' + str + '</span>';
      if (num) return '<span class="json-number">' + num + '</span>';
      if (word) return '<span class="json-word">' + word + '</span>';
      if (brack) return '<span class="json-brack">' + brack + '</span>';
      if (comma) return '<span class="json-comma">' + comma + '</span>';
      return m;
    }
  );
}
