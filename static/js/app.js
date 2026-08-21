// Watcher events that mean a session's approval/choice state may have moved,
// so an open terminal-grid tile for it should refetch its metadata.
const GRID_REFRESH_ACTIONS = new Set(['turn_complete', 'session_waiting', 'session_choice']);

// Translations for the CCSM UI. es is the default; en is the alternative.
const I18N = {
  es: {
    logout: 'salir',
    refresh: 'Refrescar',
    skin_dark: 'Oscuro',
    skin_light: 'Claro',
    skin_contrast: 'Contraste',
    skin_solarized: 'Océano',
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
    close_session: 'Archivar sesión',
    conversations: 'Conversaciones',
    switch_cards: '▦ tarjetas',
    switch_list: '≡ lista',
    search_title: 'Título y tags',
    search_text: 'Conversación y notas',
    search_hint: 'Varias palabras = todas; usa "frase exacta" entre comillas para coincidencia completa',
    searching: 'buscando...',
    no_conv: 'No se encontraron conversaciones',
    no_conv_search: 'Prueba con otros términos',
    no_conv_empty: 'Crea una sesión nueva para empezar',
    conv_origin_all: 'origen: todos',
    conv_project_all: 'proyecto: todos',
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
    scroll_for_more: 'desliza para ver más ⌄',
    loading_more: 'cargando más…',
    preview_title: 'Preview de conversación',
    you: '🧑 Tú',
    claude: '🤖 Claude',
    login_subtitle: 'Inicia sesión para continuar',
    username: 'Usuario',
    password: 'Contraseña',
    login_button: 'Entrar',
    login_totp_subtitle: 'Introduce el c\u00f3digo de tu app de autenticaci\u00f3n',
    login_totp_label: 'C\u00f3digo de 6 d\u00edgitos',
    login_totp_button: 'Verificar',
    login_totp_back: 'Volver',
    login_totp_bad: 'C\u00f3digo incorrecto o caducado',
    login_blocked: 'Demasiados intentos fallidos. Prueba dentro de un rato.',
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
    toast_session_killed: 'Sesión {0} archivada',
    toast_profile_applied: 'Perfil {0} aplicado',
    toast_profile_applied_relaunched: 'Perfil {0} aplicado; sesiones relanzadas para aplicar las nuevas credenciales: {1}',
    toast_attach_copied: 'Comando copiado: {0}',
    toast_attach_fallback: 'Comando: {0}',
    toast_error_meta: 'Error al guardar etiquetas/notas',
    toast_error_create: 'Error al crear sesión',
    toast_error_resume: 'Error al retomar',
    toast_error_conn: 'Error: {0}',
    toast_error_auth: 'Error de autenticación',
    toast_error_conn_login: 'Error de conexión',
    confirm_kill: '¿Archivar sesión {0}?',
    lan_label: '[lan]',
    name_placeholder: 'Nombre (opcional)',
    name_invalid: 'Nombre inválido: vacío o solo símbolos',
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
    cfg_2fa_on: '2FA',
    cfg_2fa_enable: 'Activar 2FA',
    cfg_2fa_disable: 'Desactivar 2FA',
    cfg_2fa_title: '2FA de {0}',
    cfg_2fa_help: 'Escanea el QR con Google Authenticator (o Aegis, 1Password\u2026) y escribe el c\u00f3digo que muestre. Hasta que no lo confirmes no se activa nada.',
    cfg_2fa_noqr: 'No se pudo dibujar el QR: a\u00f1ade la cuenta a mano con el secreto de abajo.',
    cfg_2fa_secret: 'Secreto (toca para copiar)',
    cfg_2fa_code: 'C\u00f3digo de la app',
    cfg_2fa_confirm: 'Confirmar y activar',
    cfg_2fa_enabled: '2FA activado para {0}.',
    cfg_2fa_disabled: '2FA desactivado para {0}.',
    cfg_2fa_confirm_off: '\u00bfDesactivar el 2FA de {0}?',
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
    notify_action_login_failed: 'Login fallido',
    notify_action_login_totp_failed: 'C\u00f3digo 2FA incorrecto',
    notify_action_login_blocked: 'IP bloqueada por intentos',
    notify_action_totp_enable: '2FA activado',
    notify_action_totp_disable: '2FA desactivado',
    notify_action_session_new: 'Nueva sesi\u00f3n',
    notify_action_session_kill: 'Sesi\u00f3n archivada',
    notify_action_session_resume: 'Sesi\u00f3n retomada',
    notify_action_profile_apply: 'Perfil aplicado',
    notify_action_turn_complete: 'Turno completado',
    notify_action_session_waiting: 'Esperando aprobaci\u00f3n',
    notify_action_session_choice: 'Necesita tu decisi\u00f3n',
    live_view: 'Ver sesi\u00f3n en vivo',
    live_title: 'Sesi\u00f3n {0} en vivo',
    live_closed: 'Sesi\u00f3n cerrada o desconectada.',
    live_reconnecting: 'Reconectando\u2026',
    term_grid: 'Modo terminal',
    term_grid_title: 'Modo terminal ({0})',
    term_grid_empty: 'Ninguna sesión que mostrar.',
    term_grid_pick: 'Elige una sesi\u00f3n de arriba para abrirla.',
    term_tile_minimize: 'Minimizar',
    term_tile_restore: 'Restaurar',
    term_tile_zoom: 'Pantalla completa',
    term_tile_unzoom: 'Salir de pantalla completa',
    term_tile_output: 'Contraer/expandir salida detallada (Ctrl+O)',
    term_tile_prompt: 'Escribe y pulsa Enter\u2026',
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
    chat_status_setup: 'asistente de configuración (Enter no confirma)',
    chat_approve: 'Aprobar',
    chat_approve_title: 'Aprobar el comando (opción 1)',
    chat_skip: 'Omitir',
    chat_skip_title: 'Omitir el asistente (Escape) — Enter aquí solo marca casillas',
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
    // --- voice dictation / prompt rewriting ---
    voice_dictate: 'Dictar y reescribir como prompt',
    voice_dictate_stop: 'Parar y procesar',
    voice_rewrite: 'Reescribir como prompt',
    voice_stage_recording: 'Grabando\u2026',
    voice_stage_transcribing: 'Transcribiendo\u2026',
    voice_stage_rewriting: 'Reescribiendo\u2026',
    voice_unsupported: 'Este navegador no puede dictar aqu\u00ed',
    voice_insecure: 'El dictado necesita HTTPS. Entra por https://ccsm.lan',
    voice_denied: 'Permiso de micr\u00f3fono denegado. Act\u00edvalo en los ajustes del navegador',
    voice_no_speech: 'No se ha o\u00eddo nada',
    voice_empty_input: 'No hay texto que reescribir',
    voice_err_mic: 'No se pudo acceder al micr\u00f3fono',
    voice_disabled: 'El dictado est\u00e1 desactivado en la configuraci\u00f3n',
    compose_title: 'Revisar antes de enviar',
    compose_role: 'Rol',
    compose_role_auto_detected: 'detectado',
    compose_question_title: '\u00bfNecesita aclarar algo?',
    compose_question_free: 'Tu respuesta\u2026',
    compose_answer: 'Responder',
    compose_skip: 'Omitir',
    compose_show_raw: 'Ver lo que dict\u00e9',
    compose_show_prompt: 'Ver el prompt',
    compose_raw_label: 'Transcripci\u00f3n original',
    compose_send: 'Enviar',
    compose_retry: 'Reintentar',
    compose_to_input: 'Al input',
    compose_discard: 'Descartar',
    compose_too_long: 'El prompt supera el l\u00edmite de {0} caracteres',
    compose_sent: 'Enviado a la sesi\u00f3n {0}',
    cfg_voice: 'Voz',
    cfg_voice_enabled: 'Activado',
    cfg_voice_stt_mode: 'Modo de dictado',
    cfg_voice_stt_provider: 'Proveedor de transcripci\u00f3n',
    cfg_voice_vocabulary: 'Vocabulario',
    cfg_voice_rewrite_enabled: 'Reescribir',
    cfg_voice_rewrite_provider: 'Proveedor de reescritura',
    cfg_voice_rewrite_model: 'Modelo',
    cfg_voice_default_role: 'Rol por defecto',
    cfg_voice_prompt_edit: 'Editar meta-prompt\u2026',
    cfg_voice_no_providers: 'Sin proveedores en config.yaml',
    cfg_voice_saved: 'Configuraci\u00f3n de voz guardada',
    cfg_voice_mode_whisper: 'Whisper (servidor)',
    cfg_voice_mode_webspeech: 'Navegador (Web Speech)',
    cfg_voice_mode_whisper_fallback: 'Whisper con respaldo del navegador',
    prompt_title: 'Meta-prompt de reescritura',
    prompt_save_over: 'Guardar',
    prompt_save_new: 'Guardar como\u2026',
    prompt_name_new: 'Nombre de la nueva versi\u00f3n:',
    prompt_apply: 'Aplicar esta versi\u00f3n',
    prompt_applied: 'Versi\u00f3n aplicada',
    prompt_versions: 'Versiones',
    prompt_active_label: 'Activa:',
    prompt_original_name: 'Original',
    prompt_saved: 'Versi\u00f3n guardada',
    prompt_section: 'Ir a secci\u00f3n',
    prompt_rename: 'Renombrar\u2026',
    prompt_rename_new_name: 'Nuevo nombre:',
    prompt_renamed: 'Versi\u00f3n renombrada',
    prompt_delete: 'Borrar\u2026',
    prompt_delete_confirm: '\u00bfBorrar la versi\u00f3n',
    prompt_deleted: 'Versi\u00f3n borrada',
  },
  en: {
    logout: 'logout',
    refresh: 'Refresh',
    skin_dark: 'Dark',
    skin_light: 'Light',
    skin_contrast: 'Contrast',
    skin_solarized: 'Ocean',
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
    close_session: 'Archive session',
    conversations: 'Conversations',
    switch_cards: '▦ cards',
    switch_list: '≡ list',
    search_title: 'Title & tags',
    search_text: 'Conversation & notes',
    search_hint: 'Multiple words = all; use "exact phrase" in quotes for an exact match',
    searching: 'searching...',
    no_conv: 'No conversations found',
    no_conv_search: 'Try different terms',
    no_conv_empty: 'Start a new session to get going',
    conv_origin_all: 'origin: all',
    conv_project_all: 'project: all',
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
    scroll_for_more: 'scroll for more ⌄',
    loading_more: 'loading more…',
    preview_title: 'Conversation preview',
    you: '🧑 You',
    claude: '🤖 Claude',
    login_subtitle: 'Sign in to continue',
    username: 'Username',
    password: 'Password',
    login_button: 'Sign in',
    login_totp_subtitle: 'Enter the code from your authenticator app',
    login_totp_label: '6-digit code',
    login_totp_button: 'Verify',
    login_totp_back: 'Back',
    login_totp_bad: 'Wrong or expired code',
    login_blocked: 'Too many failed attempts. Try again later.',
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
    toast_session_killed: 'Session {0} archived',
    toast_profile_applied: 'Profile {0} applied',
    toast_profile_applied_relaunched: 'Profile {0} applied; sessions relaunched to pick up the new credentials: {1}',
    toast_attach_copied: 'Command copied: {0}',
    toast_attach_fallback: 'Command: {0}',
    toast_error_meta: 'Error saving tags/notes',
    toast_error_create: 'Error creating session',
    toast_error_resume: 'Error resuming',
    toast_error_conn: 'Error: {0}',
    toast_error_auth: 'Authentication error',
    toast_error_conn_login: 'Connection error',
    confirm_kill: 'Archive session {0}?',
    lan_label: '[lan]',
    name_placeholder: 'Name (optional)',
    name_invalid: 'Invalid name: empty or only symbols',
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
    cfg_2fa_on: '2FA',
    cfg_2fa_enable: 'Enable 2FA',
    cfg_2fa_disable: 'Disable 2FA',
    cfg_2fa_title: '2FA for {0}',
    cfg_2fa_help: 'Scan the QR with Google Authenticator (or Aegis, 1Password\u2026) and type the code it shows. Nothing is enabled until you confirm.',
    cfg_2fa_noqr: 'The QR could not be drawn: add the account by hand with the secret below.',
    cfg_2fa_secret: 'Secret (tap to copy)',
    cfg_2fa_code: 'Code from the app',
    cfg_2fa_confirm: 'Confirm and enable',
    cfg_2fa_enabled: '2FA enabled for {0}.',
    cfg_2fa_disabled: '2FA disabled for {0}.',
    cfg_2fa_confirm_off: 'Disable 2FA for {0}?',
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
    notify_action_login_failed: 'Failed login',
    notify_action_login_totp_failed: 'Wrong 2FA code',
    notify_action_login_blocked: 'IP blocked for repeated failures',
    notify_action_totp_enable: '2FA enabled',
    notify_action_totp_disable: '2FA disabled',
    notify_action_session_new: 'New session',
    notify_action_session_kill: 'Session archived',
    notify_action_session_resume: 'Session resumed',
    notify_action_profile_apply: 'Profile applied',
    notify_action_turn_complete: 'Turn completed',
    notify_action_session_waiting: 'Waiting for approval',
    notify_action_session_choice: 'Decision needed',
    live_view: 'View session live',
    live_title: 'Session {0} live',
    live_closed: 'Session closed or disconnected.',
    live_reconnecting: 'Reconnecting…',
    term_grid: 'Terminal mode',
    term_grid_title: 'Terminal mode ({0})',
    term_grid_empty: 'No sessions to show.',
    term_grid_pick: 'Pick a session above to open it.',
    term_tile_minimize: 'Minimize',
    term_tile_restore: 'Restore',
    term_tile_zoom: 'Full screen',
    term_tile_unzoom: 'Exit full screen',
    term_tile_output: 'Collapse/expand detailed output (Ctrl+O)',
    term_tile_prompt: 'Type and press Enter…',
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
    chat_status_setup: 'setup wizard (Enter does not confirm)',
    chat_approve: 'Approve',
    chat_approve_title: 'Approve the command (option 1)',
    chat_skip: 'Skip',
    chat_skip_title: 'Skip the wizard (Escape) — Enter here only toggles checkboxes',
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
    // --- voice dictation / prompt rewriting ---
    voice_dictate: 'Dictate and rewrite as a prompt',
    voice_dictate_stop: 'Stop and process',
    voice_rewrite: 'Rewrite as a prompt',
    voice_stage_recording: 'Recording\u2026',
    voice_stage_transcribing: 'Transcribing\u2026',
    voice_stage_rewriting: 'Rewriting\u2026',
    voice_unsupported: 'This browser cannot dictate here',
    voice_insecure: 'Dictation needs HTTPS. Use https://ccsm.lan',
    voice_denied: 'Microphone permission denied. Enable it in the browser settings',
    voice_no_speech: 'Nothing was heard',
    voice_empty_input: 'There is no text to rewrite',
    voice_err_mic: 'Could not access the microphone',
    voice_disabled: 'Dictation is disabled in the configuration',
    compose_title: 'Review before sending',
    compose_role: 'Role',
    compose_role_auto_detected: 'detected',
    compose_question_title: 'Needs clarifying?',
    compose_question_free: 'Your answer…',
    compose_answer: 'Answer',
    compose_skip: 'Skip',
    compose_show_raw: 'Show what I said',
    compose_show_prompt: 'Show the prompt',
    compose_raw_label: 'Original transcription',
    compose_send: 'Send',
    compose_retry: 'Retry',
    compose_to_input: 'To input',
    compose_discard: 'Discard',
    compose_too_long: 'The prompt is over the {0} character limit',
    compose_sent: 'Sent to session {0}',
    cfg_voice: 'Voice',
    cfg_voice_enabled: 'Enabled',
    cfg_voice_stt_mode: 'Dictation mode',
    cfg_voice_stt_provider: 'Transcription provider',
    cfg_voice_vocabulary: 'Vocabulary',
    cfg_voice_rewrite_enabled: 'Rewrite',
    cfg_voice_rewrite_provider: 'Rewrite provider',
    cfg_voice_rewrite_model: 'Model',
    cfg_voice_default_role: 'Default role',
    cfg_voice_prompt_edit: 'Edit meta-prompt\u2026',
    cfg_voice_no_providers: 'No providers in config.yaml',
    cfg_voice_saved: 'Voice settings saved',
    cfg_voice_mode_whisper: 'Whisper (server)',
    cfg_voice_mode_webspeech: 'Browser (Web Speech)',
    cfg_voice_mode_whisper_fallback: 'Whisper with browser fallback',
    prompt_title: 'Rewriting meta-prompt',
    prompt_save_over: 'Save',
    prompt_save_new: 'Save as\u2026',
    prompt_name_new: 'Name for the new version:',
    prompt_apply: 'Apply this version',
    prompt_applied: 'Version applied',
    prompt_versions: 'Versions',
    prompt_active_label: 'Active:',
    prompt_original_name: 'Original',
    prompt_saved: 'Version saved',
    prompt_section: 'Go to section',
    prompt_rename: 'Rename…',
    prompt_rename_new_name: 'New name:',
    prompt_renamed: 'Version renamed',
    prompt_delete: 'Delete…',
    prompt_delete_confirm: 'Delete version',
    prompt_deleted: 'Version deleted',
  },
};

// Mirrors internal/sessionname's Normalize so the UI shows the same name the
// server will actually create, without a round-trip. Strips combining
// diacritics (NFD) so accented letters become ASCII, turns any other rune
// outside [A-Za-z0-9_-] into '-', collapses runs, trims, and truncates to 32.
// Returns '' when nothing valid remains (empty or only symbols).
function normalizeSessionName(raw) {
  const s = (raw || '').trim();
  if (!s) return '';
  const decomposed = s.normalize('NFD').replace(/[\u0300-\u036f]/g, '');
  let out = '';
  let lastDash = false;
  for (const ch of decomposed) {
    if (/[A-Za-z0-9_]/.test(ch)) { out += ch; lastDash = false; continue; }
    if (ch === '-') { if (!out || lastDash) continue; out += '-'; lastDash = true; continue; }
    if (!out || lastDash) continue;
    out += '-'; lastDash = true;
  }
  let name = out.replace(/-+$/, '');
  if (name.length > 32) name = name.slice(0, 32).replace(/-+$/, '');
  return name;
}

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
    // Second step of a 2FA login: shown instead of the password form once the
    // server answers `totp_required`.
    totpStep: false,
    totpCode: '',
    totpError: '',

    // Data
    sessions: { loading: false, items: [], error: '' },
    profiles: [],
    conversations: { loading: false, items: [], page: 1, hasMore: false, error: '' },

    // UI state
    lang: 'es',
    skin: 'dark',
    viewMode: 'list',
    convSearch: '',          // busqueda en titulo (q)
    convSearchText: '',      // busqueda en toda la conversacion (q_text)
    convFilters: { origin: '', project: '', from: '', to: '', alive: false },
    actionLoading: false,
    // "New session" advanced form (optional tmux name, Claude name, profile,
    // project). project defaults to "principal" (home), the historical launch.
    adv: { tmux: '', claude: '', profile: '', project: 'principal' },
    projects: [],
    preview: { open: false, messages: [], date: '', origin: '', id: '', title: '', is_alive: false, tags: '', notes: '', saving: false },
    settings: { open: false, loading: false, groups: [], editing: null, editValue: '', users: [] },
    userModal: { open: false, mode: 'add', username: '', password: '', error: '' },
    totpModal: { open: false, username: '', secret: '', uri: '', qr: '', code: '', error: '', busy: false },
    audit: { open: false, loading: false, q: '', entries: [] },
    metrics: { open: false, loading: false, data: null, model: '' },
    notify: { supported: false, permission: 'default', muted: false, es: null },
    // Voice dictation. `mode` is what the server is configured for; `effective`
    // is what this browser can actually do, which may be less (no HTTPS, no
    // MediaRecorder, no SpeechRecognition).
    voice: {
      enabled: false, mode: 'whisper_fallback', effective: '', reason: '',
      rewriteEnabled: true, roles: [], defaultRole: 'auto', maxSendLen: 16000,
      providers: [], recording: false, stage: '', target: null,
      rec: null, chunks: [], stream: null, sr: null, srText: '',
    },
    // The review panel: where a dictated or rewritten prompt lands so it can
    // be read and edited. Neither input in the app is big enough for that —
    // the chat one is a single row and the tile one is 1.4em tall.
    compose: {
      open: false, target: '', role: 'auto', detected: '', raw: '', text: '',
      // question: the one thing (if anything) the rewriter still finds
      // unclear this round; answerHistory: every question answered so far in
      // this session, sent back on each call so the model does not repeat
      // itself. See composeAnswer.
      question: null, answerHistory: [], freeAnswer: '', showRaw: false, busy: false,
    },
    promptEditor: {
      open: false, loading: false, saving: false, content: '',
      versions: [], viewing: 0,
    },
    voiceForm: null,
    live: { open: false, name: '', view: 'chat', chatStatus: '', ces: null, timer: null, msgs: [], termHist: '', meta: null, input: '', sending: false, elapsed: '', models: [], maxH: null },
    // Terminal grid: tiles keyed by session name (the same stable identity the
    // session list uses), plus a single `zoomed` name — only one tile can be
    // zoomed at a time. `narrow` mirrors the (max-width: 1023px) media query
    // (see initGridNarrowTrack): below that width the mosaic layout is
    // unusable, so tiles default to minimized and only one is ever shown at
    // a time (openGridTile/restoreTile).
    // Touch/coarse-pointer devices (phones, tablets) have no Shift key on
    // their on-screen keyboard, so Enter-to-send can't be paired with
    // Shift+Enter for a newline there: every bare Enter sent immediately,
    // splitting one multi-line message typed on a phone into several. On
    // these devices Enter always inserts a newline; sending is via the
    // button only. Desktop keyboards keep Enter=send / Shift+Enter=newline.
    coarsePointer: window.matchMedia('(pointer: coarse)').matches,
    grid: { open: false, zoomed: null, minimized: {}, tiles: {}, narrow: window.matchMedia('(max-width: 1023px)').matches },
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

    // --- Skin (color theme) ---
    // skin-init.js (loaded before Alpine, in index.html's <head>) applies
    // data-skin on <html> synchronously to avoid a flash of the wrong skin.
    // initSkin() re-applies it (idempotent) and syncs Alpine's own copy so
    // the settings menu highlights the right option.
    initSkin() {
      const saved = localStorage.getItem('ccsm_skin');
      this.skin = ['light', 'contrast', 'solarized'].includes(saved) ? saved : 'dark';
      document.documentElement.setAttribute('data-skin', this.skin);
    },
    setSkin(s) {
      this.skin = s;
      document.documentElement.setAttribute('data-skin', s);
      try { localStorage.setItem('ccsm_skin', s); } catch (e) { /* private mode */ }
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
      this.initSkin();
      this.initViewportTrack();
      this.initGridNarrowTrack();
      this.initConvInfiniteScroll();
      this.$watch('settings.open', v => this.setBodyLock());
      this.$watch('preview.open', v => this.setBodyLock());
      this.$watch('rename.open', v => this.setBodyLock());
      this.$watch('profViewer.open', v => this.setBodyLock());
      this.$watch('userModal.open', v => this.setBodyLock());
      this.$watch('totpModal.open', v => this.setBodyLock());
      this.$watch('live.open', v => this.setBodyLock());
      this.$watch('grid.open', v => this.setBodyLock());
      this.$watch('compose.open', v => this.setBodyLock());
      this.$watch('promptEditor.open', v => this.setBodyLock());
      await this.checkAuth();
      if (this.authenticated) {
        this.showLogin = false;
        this.loadAll();
        this.initNotify();
        this.initVoice();
      }
    },

    // Freezes the page behind a full-screen overlay.
    //
    // `body { overflow: hidden }` alone is NOT enough: WebKit on iOS/iPadOS
    // ignores it and keeps scrolling the document, so a swipe that reaches the
    // end of a scrollable pane inside the overlay chains to the page and drags
    // the whole thing — the overlay's header slides off the top and the panel
    // underneath shows through. Pinning the body with `position: fixed` is what
    // actually stops it there; the saved offset is restored on unlock so
    // closing a modal doesn't jump the user back to the top of the page.
    setBodyLock() {
      const lock = this.settings.open || this.preview.open || this.rename.open ||
        this.profViewer.open || this.userModal.open || this.totpModal.open ||
        this.live.open || this.grid.open || this.compose.open || this.promptEditor.open;

      if (lock) {
        if (this.scrollLock !== null) return;   // already locked by another overlay
        this.scrollLock = window.scrollY || window.pageYOffset || 0;
        document.documentElement.style.overflow = 'hidden';
        document.body.style.overflow = 'hidden';
        document.body.style.position = 'fixed';
        document.body.style.top = '-' + this.scrollLock + 'px';
        document.body.style.left = '0';
        document.body.style.right = '0';
        document.body.style.width = '100%';
        return;
      }

      if (this.scrollLock === null) return;
      const y = this.scrollLock;
      this.scrollLock = null;
      document.documentElement.style.overflow = '';
      document.body.style.overflow = '';
      document.body.style.position = '';
      document.body.style.top = '';
      document.body.style.left = '';
      document.body.style.right = '';
      document.body.style.width = '';
      window.scrollTo(0, y);
    },

    // Page offset saved while an overlay holds the scroll lock; null = unlocked.
    scrollLock: null,

    // Keeps the live modal inside the visible area when the mobile keyboard
    // opens/closes. WebKit on iOS (incl. Edge, which is WebKit there) does NOT
    // reflow position:fixed elements on the keyboard opening, so a backdrop
    // sized by dvh alone hangs at the old full height and the card, centered
    // in it, slides under the keyboard. The visualViewport is the source of
    // truth, but its height can lag behind innerHeight (and vice versa on
    // some engines), so the visible height is the min of both. Pinning the
    // overlay's height to it keeps the whole modal inside the visible area.
    pinLiveToViewport() {
      const vv = window.visualViewport;
      const vvh = (vv && vv.height) ? vv.height : window.innerHeight;
      const off = (vv && typeof vv.offsetTop === 'number' && vv.offsetTop > 0) ? vv.offsetTop : 0;
      // Floor so a transient bad read can never collapse the backdrop and
      // expose the page underneath.
      const h = Math.max(200, Math.min(vvh, window.innerHeight));
      this.live.maxH = Math.round(h * 0.8);
      const ov = this.$refs && this.$refs.liveOverlay;
      if (ov) {
        // position:fixed anchors to the LAYOUT viewport, which never shrinks.
        // When the keyboard opens, iOS pans the VISUAL viewport down (offsetTop)
        // to reveal the focused field; an overlay left at top:0 then sits above
        // the visible area and the page shows through below it. Anchor the
        // overlay to the visual viewport itself: same offset, same height, so
        // the backdrop always covers exactly what is on screen.
        ov.style.height = h + 'px';
        ov.style.top = off + 'px';
      }
    },
    initViewportTrack() {
      this.pinLiveToViewport();
      if (window.visualViewport) {
        window.visualViewport.addEventListener('resize', () => this.pinLiveToViewport());
        window.visualViewport.addEventListener('scroll', () => this.pinLiveToViewport());
      }
      window.addEventListener('resize', () => this.pinLiveToViewport());
    },

    // Keeps grid.narrow in sync with the (max-width: 1023px) breakpoint the
    // terminal grid uses to decide between the multi-tile mosaic and the
    // one-tile-at-a-time mobile behaviour (rotating a tablet can cross it
    // mid-session).
    initGridNarrowTrack() {
      const mq = window.matchMedia('(max-width: 1023px)');
      mq.addEventListener('change', () => { this.grid.narrow = mq.matches; });
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

        // A reload with a half-finished 2FA login resumes at the code step:
        // the password was already accepted, asking for it again is noise.
        if (!this.authenticated && data.totp_required) {
          this.totpStep = true;
          return;
        }

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
          this.loginPass = '';
          this.enterApp(this.loginUser);
        } else if (data.totp_required) {
          // Password accepted; the server issued a pending cookie and now
          // wants the code. Drop the password from memory either way.
          this.loginPass = '';
          this.totpCode = '';
          this.totpError = '';
          this.totpStep = true;
        } else if (resp.status === 429) {
          this.loginError = this.t('login_blocked');
        } else {
          this.loginError = this.t('toast_error_auth');
        }
      } catch (e) {
        this.loginError = this.t('toast_error_conn_login');
      }
    },

    async doTotp() {
      this.totpError = '';
      try {
        const resp = await fetch('/api/auth/totp', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ code: this.totpCode }),
        });
        const data = await resp.json();
        if (data.ok) {
          this.totpCode = '';
          this.totpStep = false;
          this.enterApp(this.loginUser);
        } else if (resp.status === 429) {
          this.totpError = this.t('login_blocked');
        } else {
          this.totpError = this.t('login_totp_bad');
        }
      } catch (e) {
        this.totpError = this.t('toast_error_conn_login');
      }
    },

    // Shared tail of both login paths (one-step and 2FA).
    enterApp(username) {
      this.authenticated = true;
      this.showLogin = false;
      this.userLabel = username;
      this.loadAll();
      this.initNotify();
    },

    // Back from the code step to the password form (wrong account, or the
    // phone isn't at hand).
    cancelTotp() {
      this.totpStep = false;
      this.totpCode = '';
      this.totpError = '';
      fetch('/api/auth/logout', { method: 'POST' });
    },

    async logout() {
      await fetch('/api/auth/logout', { method: 'POST' });
      this.authenticated = false;
      this.showLogin = true;
      this.totpStep = false;
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
      if (this.grid.open) this.syncGridTiles();
    },

    async loadConversations(page = 1) {
      this.conversations.loading = true;
      this.conversations.error = '';
      try {
        const params = new URLSearchParams({ page: String(page), per_page: '20' });
        if (this.convSearch) params.set('q', this.convSearch);
        if (this.convSearchText) params.set('q_text', this.convSearchText);
        if (this.convFilters.origin) params.set('origin', this.convFilters.origin);
        if (this.convFilters.project) params.set('project', this.convFilters.project);
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

    // Replaces a tap-to-load-more button with the standard touch-list
    // pattern: scrolling near the end of the list loads the next page on
    // its own. Both sentinels (list and card view) are observed from the
    // start — only one is ever in the DOM's layout flow at a time (the
    // other's view is x-show:none, so it never intersects), and hidden
    // (display:none) sentinels don't fire either, once conversations run
    // out. rootMargin starts the fetch a bit before the sentinel is
    // actually on screen so the next page is ready by the time the user
    // gets there.
    initConvInfiniteScroll() {
      const onIntersect = (entries) => {
        for (const e of entries) {
          if (e.isIntersecting && this.conversations.hasMore && !this.conversations.loading) {
            this.loadMoreConversations();
          }
        }
      };
      const observer = new IntersectionObserver(onIntersect, { rootMargin: '200px' });
      this.$nextTick(() => {
        if (this.$refs.convSentinelList) observer.observe(this.$refs.convSentinelList);
        if (this.$refs.convSentinelCards) observer.observe(this.$refs.convSentinelCards);
      });
    },

    async loadProfiles() {
      try {
        const resp = await fetch('/api/profiles');
        if (resp.ok) this.profiles = await resp.json();
      } catch (e) { /* ignore */ }
    },

    // Name of the profile currently applied to settings.json, taken from the
    // server's is_active flag. First match wins so the UI can never paint more
    // than one active tick even if the server ever misbehaves.
    activeProfileName() {
      const act = this.profiles.filter(p => p.is_active);
      return act.length ? act[0].name : '';
    },

    // Whether a specific profile is the active one. Single source for every
    // ✓/arrow in the UI (change-profile list and the advanced-session select),
    // so the indication is consistent across both.
    isActiveProfile(p) {
      return !!p && p.name === this.activeProfileName();
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
      this.pinLiveToViewport(); // clear any stale keyboard-anchored position
      this.live.name = s.name;
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

    // The mode/model dropdowns must always offer the session's CURRENT values
    // (meta.mode / meta.model, read from the pane) as selectable, highlighted
    // options — even when they aren't in the discovered wheel / configured
    // model list. Otherwise the browser picks the first option for display and
    // the dropdown disagrees with the status line. Prepending guarantees the
    // real value is both present and selected.
    liveModeOptions() {
      const base = this.modeOptions().slice();
      const m = this.live.meta && this.live.meta.mode;
      if (m && !base.includes(m)) base.unshift(m);
      return base;
    },
    liveModelOptions() {
      const base = this.live.models.slice();
      const m = this.live.meta && this.live.meta.model;
      if (m && !base.includes(m)) base.unshift(m);
      return base;
    },

    // NOTE: deliberately NO scrollIntoView here or in pinLiveToViewport. On
    // iOS WebKit, scrollIntoView on a field whose scroll container doesn't
    // need scrolling scrolls the DOCUMENT — which, with the body pinned
    // position:fixed by setBodyLock, pans the whole fixed layer and throws the
    // overlay off the visible area ("the overlay goes very far", page showing
    // through behind it). Correct SIZING is what keeps the field above the
    // keyboard: pinLiveToViewport anchors the overlay to the visible height,
    // so the card centers inside it and the focused input needs no scroll.
    // The delayed re-pin catches the keyboard slide-in, which can land after
    // the focus event (the visual viewport resize fires mid-animation; the
    // re-check picks up the final height regardless).
    liveFocusIn() {
      setTimeout(() => this.pinLiveToViewport(), 350);
    },

    // Changes the session mode. /mode does not exist in Claude Code (2.1.227):
    // the host resolves it by cycling the real Shift+Tab wheel, so we send the
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

    // Both tabs read off the one chat stream (termText derives from
    // termHist) — switching tabs is a pure view change, nothing to
    // start/stop.
    setLiveView(v) {
      if (v === this.live.view) return;
      this.live.view = v;
      if (v === 'term') {
        // The pane mounts fresh (x-if in index.html) with termHist already
        // in termText — the chat tab fetched it earlier — so it opens at
        // scrollTop 0, the top, unless forced down once here. Same reasoning
        // as the grid's restoreTile.
        this.$nextTick(() => {
          const el = this.$refs.livePane;
          if (el) el.scrollTop = el.scrollHeight;
        });
      }
    },

    atBottom(el) {
      return el.scrollHeight - el.scrollTop - el.clientHeight < 8;
    },

    // watchPaneResize keeps a session's tmux window matched to how many
    // characters actually fit in el (see requestPaneResize above): measures
    // once immediately, then again on every layout change via ResizeObserver
    // (window resize, grid mosaic reflow, tile zoom/minimize, sidebar toggle —
    // anything that changes el's box, without having to hook each of those
    // call sites by hand). el is torn down and recreated often (x-if/x-for on
    // the live overlay and grid tiles); stopPaneResize below and at each
    // pane's close call site keeps that from piling up stale observers.
    watchPaneResize(name, el) {
      stopPaneResize(name);
      const pre = el.querySelector('pre');
      if (!pre) return;
      const measure = () => requestPaneResize(name, el.clientWidth, el.clientHeight, pre);
      measure();
      const ro = new ResizeObserver(measure);
      ro.observe(el);
      paneResizeObservers[name] = ro;
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

    // Terminal tab: the same conversation as the Chat tab, styled as
    // terminal text (❯ prefix on user turns). Claude Code paints in tmux's
    // alternate screen (no scrollback), so this comes from the transcript,
    // not a raw tmux pane capture — plain HTML text reflows to any width on
    // its own, and this tab is read-only anyway (no input, unlike a grid
    // tile), so there's no "live current screen" to show separately from it.
    get termText() {
      return this.live.termHist || '';
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

    // The chat shows a "processing" indicator while the assistant is working:
    // the pane carries the live status line (noodling…/pondering…/…), and its
    // presence IS the working signal — it covers the whole turn (thinking, tool
    // loops, streaming) and disappears when the pane goes idle, even if a stale
    // line lingers briefly above the finished output. Waiting dialogs show
    // their own UI instead, so those hide it.
    chatWorking() {
      const m = this.live.meta;
      if (!m || !m.is_alive || !m.working || m.waiting) return false;
      return true;
    },

    // Enter sends; Shift+Enter inserts a newline. The .exact modifier is not in
    // the Alpine bundle (3.14.9): it blocks the whole listener, and Enter ended
    // up inserting a newline without sending. So the decision is made here with
    // shiftKey instead of relying on .exact.
    onChatKeydown(e) {
      if (e.shiftKey) return;          // Shift+Enter: default behaviour (newline)
      if (this.coarsePointer) return;  // touch: no Shift key, Enter is always a newline
      e.preventDefault();              // Enter: stop the newline and send
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

    // postSessionSend is the single POST /send code path, shared by the live
    // modal and the terminal grid tiles so both behave identically.
    async postSessionSend(name, body) {
      try {
        const resp = await fetch('/api/sessions/' + encodeURIComponent(name) + '/send', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
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

    async sendChatText(text) {
      if (!text || !text.trim()) return false;
      return this.postSessionSend(this.live.name, { text: text.trim() });
    },

    async sendKey(key) {
      await this.postSessionSend(this.live.name, { keys: key });
    },

    // Clicking an AskUserQuestion option: the host navigates the real picker
    // to index i itself (from its own read of the pane, not our last-known
    // selected) and confirms it — not just an Enter on whatever happened to
    // be highlighted already, which silently picked the wrong option for
    // every button except the one already selected.
    async selectChoice(i) {
      await this.postSessionSend(this.live.name, { choice: i });
      this.loadChat();
    },

    closeLive() {
      this.closeChatStream();
      this.live.open = false;
    },

    // --- Terminal grid: every active session tiled at once ---
    // Each tile is the same raw pane stream the single-session Terminal tab
    // uses (so reasoning and internal output show up, unlike the chat view),
    // asked for with colour. Metadata (approval / choice / mode) comes from
    // the session's own /chat/stream (same stream the live modal uses), NOT
    // the shared /api/events one: that one is only created inside initNotify
    // and only when the Notification API exists, so a browser without it (or
    // with a dropped /api/events connection) left tiles blind to dialogs —
    // the choice/approval bar never appeared and the pane only answered by
    // text. The per-session stream carries waiting/choice in its fingerprint
    // (handlers.ChatStream), so the tile resolves exactly like the modal.

    async openTermGrid() {
      this.grid.open = true;
      this.grid.zoomed = null;
      this.grid.minimized = {};
      if (!this.live.models.length) this.loadModels();
      await this.loadSessions();
      this.syncGridTiles();
      // Narrow (phones/small windows): openGridTile just minimized every tile.
      // With exactly one session there is nothing to choose between, so open
      // it right away instead of leaving the grid on the "pick one"
      // placeholder — two or more sessions still start all-minimized as
      // before. Only on the initial open — a later sync (the 5s poll below)
      // must not re-open a tile the user minimized on purpose.
      if (this.grid.narrow) {
        const names = Object.keys(this.grid.tiles);
        if (names.length === 1) this.restoreTile(names[0]);
      }
      // While the grid is open a new or dead session should show up quickly;
      // the usual 30s list poll would leave a stale tile on screen too long.
      if (this.pollInterval) clearInterval(this.pollInterval);
      this.pollInterval = setInterval(() => this.loadSessions(), 5000);
    },

    closeTermGrid() {
      Object.keys(this.grid.tiles).forEach(name => this.closeGridTile(name));
      this.grid.open = false;
      this.grid.zoomed = null;
      this.grid.minimized = {};
      if (this.pollInterval) clearInterval(this.pollInterval);
      this.pollInterval = setInterval(() => this.loadSessions(), 30000);
    },

    // syncGridTiles reconciles the tiles against the live session list: new
    // sessions get a tile, vanished ones lose theirs, the rest are left alone
    // so their streams are never needlessly restarted.
    syncGridTiles() {
      const live = new Set(this.sessions.items.map(s => s.name));
      live.forEach(name => { if (!this.grid.tiles[name]) this.openGridTile(name); });
      Object.keys(this.grid.tiles).forEach(name => {
        if (!live.has(name)) this.closeGridTile(name);
      });
    },

    openGridTile(name) {
      this.grid.tiles[name] = {
        name, content: '', status: '', es: null, ces: null, termHist: '',
        meta: null, input: '', sending: false,
      };
      // Narrow screens can't fit a mosaic, so every tile starts minimized —
      // the user picks one at a time from the header chips (restoreTile).
      if (this.grid.narrow) this.grid.minimized[name] = true;
      this.startTileStream(name);
      this.startTileChatStream(name);
      this.fetchTileMeta(name);
    },

    // tileText mirrors the single-session termText getter: the /chat-derived
    // history (termHist) so there's something to scroll up to, then a
    // separator, then the live (optionally coloured) pane screen.
    tileText(name) {
      const tile = this.grid.tiles[name];
      if (!tile) return '';
      const live = ansiToHtml(stripInputBoxChrome(tile.content || ''));
      if (!tile.termHist) return live;
      return escapeHtml(tile.termHist) + '\n\n' + escapeHtml(this.t('live_screen_sep')) + '\n' + live;
    },

    closeGridTile(name) {
      this.stopTileStream(name);
      this.stopTileChatStream(name);
      stopPaneResize(name);
      delete this.grid.minimized[name];
      if (this.grid.zoomed === name) this.grid.zoomed = null;
      delete this.grid.tiles[name];
    },

    startTileStream(name) {
      const tile = this.grid.tiles[name];
      if (!tile) return;
      const es = new EventSource('/api/sessions/' + encodeURIComponent(name) + '/stream?color=1');
      tile.es = es;
      es.onopen = () => { tile.status = ''; };
      es.onmessage = (ev) => {
        const el = document.getElementById('tile-pane-' + name);
        const stick = el ? this.atBottom(el) : true;
        tile.content = ev.data.replace(/\\n/g, '\n');
        tile.status = '';
        this.$nextTick(() => { if (stick && el) el.scrollTop = el.scrollHeight; });
      };
      // Don't close() on error: the browser reconnects on its own, and closing
      // here would kill that retry. Same reasoning as startTileChatStream below.
      es.onerror = () => { tile.status = this.t('live_reconnecting'); };
    },

    stopTileStream(name) {
      const tile = this.grid.tiles[name];
      if (tile && tile.es) { tile.es.close(); tile.es = null; }
    },

    // The tile's /chat/stream pushes the full session payload (messages,
    // waiting, choice, modes) the moment it changes — the same stream the
    // live modal uses. Keeping it per tile means the choice/approval bar
    // shows up when the dialog opens and disappears when it resolves, even
    // if the shared /api/events stream (initNotify, Notification-gated) is
    // dead in this browser.
    startTileChatStream(name) {
      const tile = this.grid.tiles[name];
      if (!tile) return;
      const es = new EventSource('/api/sessions/' + encodeURIComponent(name) + '/chat/stream');
      tile.ces = es;
      es.onopen = () => { if (this.grid.tiles[name]) this.grid.tiles[name].status = ''; };
      es.onmessage = (ev) => {
        const t = this.grid.tiles[name];
        if (!t) return;
        try {
          this.applyTileChat(name, JSON.parse(ev.data));
        } catch (e) { /* keep the last good meta */ }
      };
      // Don't close() on error: EventSource reconnects on its own.
      es.onerror = () => {
        if (this.grid.tiles[name]) this.grid.tiles[name].status = this.t('live_reconnecting');
      };
    },

    stopTileChatStream(name) {
      const tile = this.grid.tiles[name];
      if (tile && tile.ces) { tile.ces.close(); tile.ces = null; }
    },

    // applyTileChat folds one /chat payload into a tile: meta (approval/choice/
    // mode, what the bar and status chrome read) AND termHist (the "history"
    // text above the live screen). Claude Code draws its TUI on the alternate
    // screen, so tmux's scrollback for that pane is just the current screen —
    // the /chat messages are the only scrollable history, same reasoning as
    // the single-session Terminal tab. Both the tile's own /chat/stream and
    // fetchTileMeta go through here so the historical part stays current even
    // when the shared /api/events stream is dead (the iOS case).
    applyTileChat(name, d) {
      const t = this.grid.tiles[name];
      if (!t) return; // tile closed while in flight
      t.meta = d;
      const msgs = (d.messages || []).map(m => String(m.content || '').trim() ? (m.role === 'user' ? '❯ ' : '') + String(m.content).trim() : '');
      t.termHist = msgs.filter(Boolean).join('\n\n');
      const el = document.getElementById('tile-pane-' + name);
      if (el && this.atBottom(el)) this.$nextTick(() => { el.scrollTop = el.scrollHeight; });
    },

    // fetchTileMeta pulls the approval/choice/mode state for one tile. Called
    // when the tile opens and whenever /api/events reports that session moved.
    async fetchTileMeta(name) {
      const tile = this.grid.tiles[name];
      if (!tile) return;
      try {
        const resp = await fetch('/api/sessions/' + encodeURIComponent(name) + '/chat');
        if (!resp.ok) return;
        this.applyTileChat(name, await resp.json());
      } catch (e) { /* transient; the next event retries */ }
    },

    tileRcConnected(name) {
      const s = this.sessions.items.find(x => x.name === name);
      return !!s && s.status === 'rc_connected';
    },

    tileTask(name) {
      const s = this.sessions.items.find(x => x.name === name);
      return (s && s.task) || '';
    },

    // Mirrors modeOptions(): once the host has calibrated that session's real
    // Shift+Tab wheel it's in tile.meta.modes; before that, the standard four.
    tileModeOptions(name) {
      const tile = this.grid.tiles[name];
      const w = tile && tile.meta && tile.meta.modes;
      if (w && w.length) return w;
      return ['auto', 'plan', 'accept-edits', 'manual'];
    },

    // Current mode/model of a tile (empty until the session's been calibrated).
    // The header selects show these instead of a fixed "Modo"/"Modelo"
    // placeholder, so the active values stay visible without opening the
    // dropdown — and the header keeps its single row with no separate
    // mode/model text.
    tileMetaMode(name) {
      const t = this.grid.tiles[name];
      return t && t.meta && t.meta.mode ? t.meta.mode : '';
    },
    tileMetaModel(name) {
      const t = this.grid.tiles[name];
      return t && t.meta && t.meta.model ? t.meta.model : '';
    },

    // Fit a header select to the value it is showing. width:auto would size to
    // the WIDEST option, not the selected one, so the width is derived from the
    // value's length instead — capped so a long model name can only push the
    // header into its horizontal scroll (the documented fallback), never wrap
    // it onto a second line. Empty value → '' → the CSS width applies.
    tileSelectWidth(v) {
      return v ? 'width:' + Math.min(Math.max(v.length * 0.32 + 1.4, 3.75), 8) + 'rem' : '';
    },

    async setTileMode(name, mode) {
      if (!mode) return;
      if (await this.postSessionSend(name, { mode })) {
        this.toastSuccess(this.t('mode_sent', [mode]));
        this.fetchTileMeta(name);
      }
    },

    async setTileModel(name, model) {
      if (!model) return;
      if (await this.postSessionSend(name, { text: '/model ' + model })) {
        this.toastSuccess(this.t('model_sent', [model]));
        this.fetchTileMeta(name);
      }
    },

    visibleTileNames() {
      return Object.keys(this.grid.tiles).filter(n => !this.grid.minimized[n]);
    },

    minimizedTileNames() {
      return Object.keys(this.grid.minimized);
    },

    minimizeTile(name) {
      this.grid.minimized[name] = true;
      if (this.grid.zoomed === name) this.grid.zoomed = null;
    },

    restoreTile(name) {
      // Narrow: at most one tile visible at a time (there's no room for a
      // mosaic), so opening one minimizes whichever was open before it.
      if (this.grid.narrow) {
        Object.keys(this.grid.tiles).forEach(n => { if (n !== name) this.grid.minimized[n] = true; });
      }
      delete this.grid.minimized[name];
      // A minimized tile isn't in the DOM at all (see the x-if guard in
      // index.html), so it mounts fresh here with whatever content already
      // streamed in while hidden — at scrollTop 0, i.e. the top, not the
      // bottom the startTileStream/fetchTileMeta "stick" logic only
      // maintains once the pane already exists. Force it down once here.
      this.$nextTick(() => {
        const el = document.getElementById('tile-pane-' + name);
        if (el) el.scrollTop = el.scrollHeight;
      });
    },

    // Only one tile can be zoomed, so this is a single field rather than a
    // per-tile flag: the "two zoomed at once" state cannot be represented.
    toggleTileZoom(name) {
      if (this.grid.minimized[name]) return;
      this.grid.zoomed = this.grid.zoomed === name ? null : name;
    },

    // Mirrors scrollLive(): dir -1 scrolls up one page, dir 1 goes to the end.
    scrollTilePane(name, dir) {
      const el = document.getElementById('tile-pane-' + name);
      if (!el) return;
      if (dir > 0) {
        el.scrollTop = el.scrollHeight;
      } else {
        el.scrollTop = Math.max(0, el.scrollTop - el.clientHeight);
      }
    },

    // tileRows packs the visible tiles into near-square rows so an incomplete
    // last row never leaves an empty gap: each row is its own flex line where
    // every tile gets flex:1, so a short row's tiles simply grow to fill it
    // (a lone tile in the last row takes the full row width automatically —
    // no per-tile width math needed). Row sizes differ by at most one tile,
    // e.g. 3 tiles -> [2,1] (not a 2x2 grid with a dead cell), 5 -> [3,2],
    // 7 -> [3,2,2]. Recomputed on every render, driven only by tile count, so
    // minimizing a tile immediately reflows the rest into fewer/larger rows.
    tileRows() {
      const names = this.visibleTileNames();
      const n = names.length;
      if (n === 0) return [];
      const cols = Math.ceil(Math.sqrt(n));
      const rows = Math.ceil(n / cols);
      const base = Math.floor(n / rows);
      let extra = n % rows;
      const result = [];
      let i = 0;
      for (let r = 0; r < rows; r++) {
        const size = base + (extra > 0 ? 1 : 0);
        if (extra > 0) extra--;
        result.push(names.slice(i, i + size));
        i += size;
      }
      return result;
    },

    async sendTileText(name) {
      const tile = this.grid.tiles[name];
      if (!tile || !tile.input.trim()) return;
      const text = tile.input.trim();
      tile.input = '';
      tile.sending = true;
      if (await this.postSessionSend(name, { text })) this.fetchTileMeta(name);
      tile.sending = false;
    },

    async sendTileKey(name, key) {
      if (await this.postSessionSend(name, { keys: key })) this.fetchTileMeta(name);
    },

    // Tile mirror of selectChoice: clicking an AskUserQuestion option posts
    // its own index, not a blind Enter — the host navigates the real picker
    // there and confirms it (host.sessionChoice). The modal had this; the
    // grid tile's options were inert spans until they were made buttons, so
    // "responder con las opciones" in Modo terminal did nothing.
    async selectTileChoice(name, i) {
      if (await this.postSessionSend(name, { choice: i })) this.fetchTileMeta(name);
    },

    onTileKeydown(e, name) {
      if (e.shiftKey) return;          // Shift+Enter: newline
      if (this.coarsePointer) return;  // touch: no Shift key, Enter is always a newline
      e.preventDefault();              // Enter: send
      this.sendTileText(name);
    },

    // Tile prompt input: while focused it opens up to three lines and takes the
    // whole bar width on its own row (the controls drop to one compact row
    // beneath — .tgrid-prompt:focus-within in terminal-grid.css). Growing is JS
    // (scrollHeight, capped well below the live chat's 128px); the CSS
    // min-height holds the three lines while the box is empty. On blur the
    // inline height is cleared so a long prompt doesn't leave the bar
    // permanently tall after the input loses focus.
    tileInputGrow(el) {
      if (!el) return;
      el.style.height = 'auto';
      el.style.height = Math.min(el.scrollHeight, 84) + 'px';
    },
    tileInputFocus(el) { this.tileInputGrow(el); },
    tileInputBlur(el) { if (el) el.style.height = ''; },

    // --- Settings panel ---
    async openSettings() {
      this.settings.open = true;
      // The voice form is rebuilt on every open even when the generated groups
      // are cached: it is editable, so a stale copy would show the previous
      // provider after someone changed it from another device.
      if (this.settings.groups.length && this.voiceForm) return;
      this.settings.loading = true;
      try {
        const resp = await fetch('/api/config');
        if (!resp.ok) throw new Error('config unavailable');
        const cfg = await resp.json();
        this.settings.groups = this.buildConfigGroups(cfg);
        this.voiceFormInit(cfg);
        this.applyVoiceConfig(cfg.voice);
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
          // Refresh the matching grid tile's approval/choice state. For these
          // three watcher actions the `user` field carries the SESSION name,
          // not a login name (see internal/server/watcher.go) — don't "fix" it.
          if (this.grid.open && GRID_REFRESH_ACTIONS.has(ev.action) && this.grid.tiles[ev.user]) {
            this.fetchTileMeta(ev.user);
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
      if (action === 'login_failed' || action === 'login_totp_failed' || action === 'login_blocked') return 'bg-danger/15 text-danger';
      if (action.startsWith('session_')) return 'bg-accent/15 text-accent';
      if (action.startsWith('user_') || action.startsWith('totp_') || action === 'config_update') return 'bg-warn/15 text-warn';
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
          await this.reloadUsers();
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
          await this.reloadUsers();
        } else {
          this.toastMsg(data.error || 'Error', 'error');
        }
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
      }
    },

    // --- Two-factor authentication (TOTP) ---

    // The QR encoder is ~55 KB and only ever needed while enrolling, so it is
    // fetched on demand instead of loading on every page view. Same-origin, so
    // the CSP's script-src 'self' allows the injected tag.
    loadQrLib() {
      if (window.qrcode) return Promise.resolve(true);
      if (!this._qrLib) {
        this._qrLib = new Promise((resolve) => {
          const s = document.createElement('script');
          s.src = '/static/js/qrcode.js';
          s.onload = () => resolve(true);
          s.onerror = () => resolve(false);   // fall back to the printed secret
          document.head.appendChild(s);
        });
      }
      return this._qrLib;
    },

    async openTotpEnroll(username) {
      this.totpModal = { open: true, username, secret: '', uri: '', qr: '', code: '', error: '', busy: false };
      try {
        const resp = await fetch('/api/config/users/' + encodeURIComponent(username) + '/totp', { method: 'POST' });
        const data = await resp.json();
        if (!resp.ok) {
          this.totpModal.error = data.error || 'Error';
          return;
        }
        this.totpModal.secret = data.secret;
        this.totpModal.uri = data.uri;
        if (await this.loadQrLib()) {
          // Type 0 = pick the smallest version that fits; L is plenty for a
          // screen-to-camera scan. SVG, not the img/data-URL variant: the CSP
          // has no img-src, so a data: URL would be blocked by default-src.
          // NOT scalable:true — that drops width/height and the <svg> collapses
          // to a zero-size box inside the flex container.
          const qr = window.qrcode(0, 'L');
          qr.addData(data.uri);
          qr.make();
          this.totpModal.qr = qr.createSvgTag({ cellSize: 4, margin: 2 });
        }
      } catch (e) {
        this.totpModal.error = this.t('toast_error_conn', [e.message]);
      }
    },

    async confirmTotp() {
      this.totpModal.error = '';
      if (!this.totpModal.code) return;
      this.totpModal.busy = true;
      try {
        const resp = await fetch('/api/config/users/' + encodeURIComponent(this.totpModal.username) + '/totp', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ code: this.totpModal.code }),
        });
        const data = await resp.json();
        if (data.ok) {
          this.totpModal.open = false;
          this.toastMsg(this.t('cfg_2fa_enabled', [this.totpModal.username]), 'success');
          await this.reloadUsers();
        } else {
          this.totpModal.error = data.error || 'Error';
        }
      } catch (e) {
        this.totpModal.error = this.t('toast_error_conn', [e.message]);
      }
      this.totpModal.busy = false;
    },

    async disableTotp(username) {
      if (!confirm(this.t('cfg_2fa_confirm_off', [username]))) return;
      try {
        const resp = await fetch('/api/config/users/' + encodeURIComponent(username) + '/totp', { method: 'DELETE' });
        const data = await resp.json();
        if (data.ok) {
          this.toastMsg(this.t('cfg_2fa_disabled', [username]), 'success');
          await this.reloadUsers();
        } else {
          this.toastMsg(data.error || 'Error', 'error');
        }
      } catch (e) {
        this.toastMsg(this.t('toast_error_conn', [e.message]), 'error');
      }
    },

    async reloadUsers() {
      try {
        const r = await fetch('/api/config/users');
        if (r.ok) this.settings.users = await r.json();
      } catch (e) { /* the panel keeps the previous list */ }
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
      const claude = this.adv.claude.trim();
      let tmux = '';
      if (this.adv.tmux.trim()) {
        tmux = normalizeSessionName(this.adv.tmux);
        if (!tmux) {
          this.toastMsg(this.t('name_invalid'), 'error');
          return;
        }
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
          const data = await resp.json();
          if (data.relaunched && data.relaunched.length) {
            this.toastMsg(this.t('toast_profile_applied_relaunched', [name, data.relaunched.join(', ')]), 'success');
            await this.loadSessions();
          } else {
            this.toastMsg(this.t('toast_profile_applied', [name]), 'success');
          }
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
      const newName = normalizeSessionName(this.rename.tmuxName);
      if (!newName) {
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

    // --- Voice: dictation and prompt rewriting ---

    // initVoice mirrors initNotify: read what the server offers, then work out
    // what THIS browser can actually do. Both are needed — a server with
    // whisper configured is useless in a browser that cannot record, and a
    // page served over plain HTTP cannot use the microphone at all.
    async initVoice() {
      try {
        const resp = await fetch('/api/config');
        if (!resp.ok) return;
        const cfg = await resp.json();
        this.applyVoiceConfig(cfg.voice);
      } catch (e) { /* voice stays off; nothing else depends on it */ }
    },

    applyVoiceConfig(v) {
      if (!v) return;
      this.voice.enabled = !!v.enabled;
      this.voice.mode = (v.stt && v.stt.mode) || 'whisper_fallback';
      this.voice.rewriteEnabled = !!(v.rewrite && v.rewrite.enabled);
      this.voice.defaultRole = (v.rewrite && v.rewrite.default_role) || 'auto';
      this.voice.maxSendLen = v.max_send_len || 16000;
      this.voice.providers = v.providers || [];
      this.voice.effective = this.resolveVoiceMode();
      this.compose.role = this.voice.defaultRole;
      this.loadVoiceRoles();
    },

    // resolveVoiceMode decides how this browser will capture speech, and
    // records why when it cannot.
    //
    // Order matters: the secure-context check comes first because without it
    // both APIs are simply absent, and reporting "unsupported browser" for
    // what is really "you are on http://" sends people debugging the wrong
    // thing.
    resolveVoiceMode() {
      this.voice.reason = '';
      if (!this.voice.enabled) { this.voice.reason = 'voice_disabled'; return ''; }
      if (!window.isSecureContext) { this.voice.reason = 'voice_insecure'; return ''; }

      const canRecord = !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia && window.MediaRecorder);
      const canSpeech = !!(window.SpeechRecognition || window.webkitSpeechRecognition);
      const hasSTT = this.voice.providers.some(p => p.stt);

      if (this.voice.mode === 'webspeech') {
        if (canSpeech) return 'webspeech';
        this.voice.reason = 'voice_unsupported';
        return '';
      }
      if (this.voice.mode === 'whisper') {
        if (canRecord && hasSTT) return 'whisper';
        this.voice.reason = 'voice_unsupported';
        return '';
      }
      // whisper_fallback
      if (canRecord && hasSTT) return 'whisper';
      if (canSpeech) return 'webspeech';
      this.voice.reason = 'voice_unsupported';
      return '';
    },

    // The mic button only appears where dictation can work; the sparkle button
    // needs no audio at all, so it appears whenever rewriting is configured.
    canDictate() { return !!this.voice.effective; },
    canRewrite() { return this.voice.enabled && this.voice.rewriteEnabled; },

    async loadVoiceRoles() {
      if (!this.voice.enabled) return;
      try {
        const resp = await fetch('/api/voice/prompt');
        if (!resp.ok) return;
        const data = await resp.json();
        this.voice.roles = data.roles || [];
      } catch (e) { /* the dropdown just stays empty */ }
    },

    roleLabel(id) {
      const r = this.voice.roles.find(x => x.id === id);
      if (!r) return id;
      return this.lang === 'en' ? r.en : r.es;
    },

    // --- Recording ---

    // pickAudioMime asks the browser what it can actually record. Safari on
    // iOS produces audio/mp4 (AAC) and everything else webm/opus, so hardcoding
    // webm silently breaks dictation on iPhone.
    pickAudioMime() {
      const candidates = [
        'audio/webm;codecs=opus', 'audio/webm',
        'audio/mp4;codecs=mp4a.40.2', 'audio/mp4',
        'audio/ogg;codecs=opus', 'audio/ogg',
      ];
      for (const c of candidates) {
        if (window.MediaRecorder && MediaRecorder.isTypeSupported && MediaRecorder.isTypeSupported(c)) return c;
      }
      return '';
    },

    toggleVoice(target) {
      if (this.voice.recording) { this.stopVoice(); return; }
      this.startVoice(target);
    },

    async startVoice(target) {
      if (!this.canDictate()) {
        this.toastError(this.t(this.voice.reason || 'voice_unsupported'));
        return;
      }
      this.voice.target = target;
      if (this.voice.effective === 'webspeech') return this.startWebSpeech();
      return this.startRecorder();
    },

    stopVoice() {
      if (this.voice.sr) { try { this.voice.sr.stop(); } catch (e) {} return; }
      if (this.voice.rec && this.voice.rec.state !== 'inactive') { this.voice.rec.stop(); }
    },

    async startRecorder() {
      let stream;
      try {
        stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      } catch (e) {
        this.toastError(this.t(e && e.name === 'NotAllowedError' ? 'voice_denied' : 'voice_err_mic'));
        return;
      }
      const mime = this.pickAudioMime();
      let rec;
      try {
        rec = mime ? new MediaRecorder(stream, { mimeType: mime }) : new MediaRecorder(stream);
      } catch (e) {
        stream.getTracks().forEach(t => t.stop());
        this.toastError(this.t('voice_err_mic'));
        return;
      }
      this.voice.chunks = [];
      this.voice.stream = stream;
      this.voice.rec = rec;
      rec.ondataavailable = e => { if (e.data && e.data.size) this.voice.chunks.push(e.data); };
      rec.onstop = () => this.finishRecording();
      rec.start();
      this.voice.recording = true;
      this.voice.stage = 'voice_stage_recording';
    },

    async finishRecording() {
      const rec = this.voice.rec;
      this.voice.recording = false;
      if (this.voice.stream) {
        // Release the mic promptly: leaving the track live keeps the browser's
        // recording indicator on and, on mobile, holds the audio session.
        this.voice.stream.getTracks().forEach(t => t.stop());
        this.voice.stream = null;
      }
      this.voice.rec = null;
      const chunks = this.voice.chunks;
      this.voice.chunks = [];
      if (!chunks.length) { this.voice.stage = ''; this.toastError(this.t('voice_no_speech')); return; }

      const type = (rec && rec.mimeType) || chunks[0].type || 'audio/webm';
      const blob = new Blob(chunks, { type });
      this.voice.stage = 'voice_stage_transcribing';
      try {
        const resp = await fetch('/api/voice/transcribe', {
          method: 'POST',
          headers: { 'Content-Type': type },
          body: blob,
        });
        const data = await resp.json().catch(() => ({}));
        if (!resp.ok) { throw new Error(data.error || 'transcription failed'); }
        await this.afterTranscript(data.text || '');
      } catch (e) {
        this.voice.stage = '';
        this.toastError(e.message);
      }
    },

    // Web Speech runs entirely in the browser; there is no audio upload and no
    // server round trip for this half.
    startWebSpeech() {
      const SR = window.SpeechRecognition || window.webkitSpeechRecognition;
      const sr = new SR();
      sr.lang = this.lang === 'en' ? 'en-US' : 'es-ES';
      sr.interimResults = false;
      sr.continuous = false;
      this.voice.srText = '';
      sr.onresult = e => {
        for (let i = e.resultIndex; i < e.results.length; i++) {
          if (e.results[i].isFinal) this.voice.srText += e.results[i][0].transcript;
        }
      };
      sr.onerror = e => {
        this.voice.recording = false;
        this.voice.stage = '';
        this.voice.sr = null;
        this.toastError(this.t(e && e.error === 'not-allowed' ? 'voice_denied' : 'voice_err_mic'));
      };
      sr.onend = async () => {
        this.voice.recording = false;
        this.voice.sr = null;
        const text = (this.voice.srText || '').trim();
        if (!text) { this.voice.stage = ''; this.toastError(this.t('voice_no_speech')); return; }
        await this.afterTranscript(text);
      };
      try { sr.start(); } catch (e) { this.toastError(this.t('voice_err_mic')); return; }
      this.voice.sr = sr;
      this.voice.recording = true;
      this.voice.stage = 'voice_stage_recording';
    },

    // afterTranscript is where the two stages join: the mic button always
    // rewrites, because dictating without rewriting is exactly what this
    // feature exists to avoid. When rewriting is off or unconfigured it
    // degrades to opening the panel with the raw text, which is still better
    // than dropping several thousand characters into a one-row input.
    async afterTranscript(text) {
      const target = this.voice.target;
      if (!this.canRewrite()) {
        this.voice.stage = '';
        this.openCompose(target, text, { role: '', prompt: text, question: null });
        return;
      }
      this.voice.stage = 'voice_stage_rewriting';
      const res = await this.callRewrite(text, this.voice.defaultRole, []);
      this.voice.stage = '';
      if (res) this.openCompose(target, text, res);
    },

    // The sparkle button: rewrite whatever is already in the input, wherever
    // it came from — the system keyboard's own dictation, or typing.
    async runRewrite(target) {
      const text = (this.inputFor(target) || '').trim();
      if (!text) { this.toastError(this.t('voice_empty_input')); return; }
      this.voice.stage = 'voice_stage_rewriting';
      const res = await this.callRewrite(text, this.voice.defaultRole, []);
      this.voice.stage = '';
      if (res) this.openCompose(target, text, res);
    },

    async callRewrite(text, role, answers) {
      try {
        const resp = await fetch('/api/voice/rewrite', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ text: text, role: role, answers: answers || [] }),
        });
        const data = await resp.json().catch(() => ({}));
        if (!resp.ok) throw new Error(data.error || 'rewrite failed');
        return data;
      } catch (e) {
        this.toastError(e.message);
        return null;
      }
    },

    // --- Review panel ---

    // inputFor / setInput are the only places that know where a target's text
    // lives, so the chat and a grid tile are handled by one set of functions.
    inputFor(target) {
      if (target === 'chat') return this.live.input;
      const tile = this.grid.tiles[target];
      return tile ? tile.input : '';
    },
    setInput(target, value) {
      if (target === 'chat') { this.live.input = value; return; }
      if (this.grid.tiles[target]) this.grid.tiles[target].input = value;
    },
    composeSession() {
      return this.compose.target === 'chat' ? this.live.name : this.compose.target;
    },

    openCompose(target, raw, res) {
      this.compose.open = true;
      this.compose.target = target;
      this.compose.raw = raw;
      this.compose.text = res.prompt || raw;
      this.compose.detected = res.role || '';
      this.compose.role = res.role || this.voice.defaultRole;
      this.compose.question = res.question || null;
      this.compose.answerHistory = [];
      this.compose.freeAnswer = '';
      this.compose.showRaw = false;
      this.compose.busy = false;
    },

    closeCompose() {
      this.compose.open = false;
      this.compose.question = null;
      this.compose.answerHistory = [];
      this.compose.freeAnswer = '';
      this.compose.raw = '';
      this.compose.text = '';
    },

    // Answering re-runs the rewrite with every answer so far folded in, and
    // the reply may itself carry another question — the clarification is a
    // loop of one question per round, not a single batch, so this both
    // answers the current one and asks for the next. The server's own round
    // cap (maxClarifyRounds) is what stops this from bouncing forever, not
    // anything client-side.
    //
    // choice is the option button's label, when one was clicked; undefined
    // means the free-text field was used instead (composeAnswer() with no
    // argument, from the input's Enter or the Responder button).
    async composeAnswer(choice) {
      const q = this.compose.question;
      if (!q) return;
      const answer = (choice != null ? choice : this.compose.freeAnswer).trim();
      if (!answer) { this.composeSkip(); return; }
      this.compose.answerHistory.push({ question: q.text, answer: answer });
      this.compose.freeAnswer = '';
      this.compose.busy = true;
      const res = await this.callRewrite(this.compose.raw, this.compose.role, this.compose.answerHistory);
      this.compose.busy = false;
      if (!res) { this.compose.answerHistory.pop(); return; }
      this.compose.text = res.prompt || this.compose.text;
      this.compose.detected = res.role || this.compose.detected;
      this.compose.question = res.question || null;
    },

    // Skipping drops the current question without answering it and asks
    // nothing further: the provisional prompt above is already the model's
    // best attempt, so there is nothing to re-run.
    composeSkip() {
      this.compose.question = null;
    },

    // Retry rewrites from the ORIGINAL transcription, not from the edited
    // prompt: that is what makes changing the role in the header useful, and
    // it avoids compounding the model's own output turn after turn.
    async composeRetry() {
      if (!this.compose.raw) return;
      this.compose.busy = true;
      const res = await this.callRewrite(this.compose.raw, this.compose.role, []);
      this.compose.busy = false;
      if (!res) return;
      this.compose.text = res.prompt || this.compose.text;
      this.compose.detected = res.role || '';
      this.compose.question = res.question || null;
      this.compose.answerHistory = [];
    },

    composeCount() { return (this.compose.text || '').length; },
    composeOver() { return this.composeCount() > this.voice.maxSendLen; },
    composeNearLimit() { return this.composeCount() > this.voice.maxSendLen * 0.9; },

    composeToInput() {
      this.setInput(this.compose.target, this.compose.text);
      this.closeCompose();
    },

    // Sending reuses postSessionSend, the single POST /send code path the chat
    // and the tiles already share — so the panel adds a caller, not a second
    // way to send. It is called directly rather than through sendChat() /
    // sendTileText() because those return nothing, and the panel must stay
    // open (with the text intact) when the send fails.
    async composeSend() {
      if (this.composeOver()) {
        this.toastError(this.t('compose_too_long', [this.voice.maxSendLen]));
        return;
      }
      const target = this.compose.target;
      const text = (this.compose.text || '').trim();
      if (!text) return;
      const session = this.composeSession();
      this.compose.busy = true;
      const ok = await this.postSessionSend(session, { text: text });
      this.compose.busy = false;
      if (!ok) return;   // postSessionSend has already explained why
      if (target === 'chat') { this.loadChat(); } else { this.fetchTileMeta(target); }
      this.setInput(target, '');
      this.toastSuccess(this.t('compose_sent', [session]));
      this.closeCompose();
    },

    // --- Voice settings ---

    voiceFormInit(cfg) {
      const v = (cfg && cfg.voice) || {};
      this.voiceForm = {
        enabled: !!v.enabled,
        mode: (v.stt && v.stt.mode) || 'whisper_fallback',
        modes: (v.stt && v.stt.modes) || ['whisper', 'webspeech', 'whisper_fallback'],
        sttProvider: (v.stt && v.stt.provider) || '',
        vocabulary: (v.stt && v.stt.vocabulary) || '',
        rewriteEnabled: !!(v.rewrite && v.rewrite.enabled),
        rewriteProvider: (v.rewrite && v.rewrite.provider) || '',
        model: (v.rewrite && v.rewrite.model) || '',
        defaultRole: (v.rewrite && v.rewrite.default_role) || 'auto',
        providers: v.providers || [],
      };
    },

    async saveVoiceSettings() {
      const f = this.voiceForm;
      if (!f) return;
      const body = {
        voice: {
          enabled: f.enabled,
          stt: { mode: f.mode, provider: f.sttProvider, vocabulary: f.vocabulary },
          rewrite: {
            enabled: f.rewriteEnabled, provider: f.rewriteProvider, model: f.model,
            default_role: f.defaultRole,
          },
        },
      };
      try {
        const resp = await fetch('/api/config', {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        });
        const data = await resp.json().catch(() => ({}));
        if (!resp.ok) throw new Error(data.error || 'error');
        this.toastSuccess(this.t('cfg_voice_saved'));
        // Re-read rather than trusting the local form: the server is what
        // decides, and a mode that this browser cannot honour must be
        // reflected in the buttons immediately.
        const fresh = await fetch('/api/config');
        if (fresh.ok) {
          const cfg = await fresh.json();
          this.applyVoiceConfig(cfg.voice);
          this.voiceFormInit(cfg);
        }
      } catch (e) {
        this.toastError(e.message);
      }
    },

    // --- Meta-prompt editor ---
    //
    // Versions are named and independent of which one is active: saving
    // (over the one being viewed, or as a brand-new one) and applying (which
    // only moves the active pointer) are two separate actions. The original
    // shipped with the project is version id 0 and can never be overwritten,
    // only viewed, applied, or edited-and-saved-as-new.

    async openPromptEditor() {
      this.promptEditor.open = true;
      this.promptEditor.loading = true;
      try {
        const resp = await fetch('/api/voice/prompt');
        if (!resp.ok) throw new Error('cannot read the meta-prompt');
        const data = await resp.json();
        this.promptEditor.content = data.content || '';
        this.promptEditor.versions = data.versions || [];
        this.voice.roles = data.roles || this.voice.roles;
        // Open on whichever version is actually active — that is what
        // dictation is using right now, not always the original.
        const active = this.promptActiveVersion();
        this.promptEditor.viewing = active ? active.id : 0;
      } catch (e) {
        this.toastError(e.message);
      }
      this.promptEditor.loading = false;
    },

    // Label for one dropdown option. Which version is actually in effect is
    // shown separately, by the "Activa: X" badge in the header — this used to
    // also mark it with a check mark here, which only doubled up with the
    // select's own native indicator for whichever option is currently chosen
    // (i.e. loaded for viewing/editing), so it was dropped.
    promptVersionLabel(v) {
      return v.original ? this.t('prompt_original_name') : v.name;
    },

    // Shared lookups so the actions below stay one-liners instead of each
    // repeating the same find() over promptEditor.versions.
    promptVersionById(id) {
      return (this.promptEditor.versions || []).find(v => v.id === id) || null;
    },

    promptActiveVersion() {
      return (this.promptEditor.versions || []).find(v => v.active) || null;
    },

    promptViewingIsOriginal() {
      return this.promptEditor.viewing === 0;
    },

    promptViewingIsActive() {
      const v = this.promptVersionById(this.promptEditor.viewing);
      return !!(v && v.active);
    },

    // The section list drives a jump menu. Editing a 4 KB document in a phone
    // textarea is the case this exists for.
    promptSections() {
      const out = [];
      const re = /^#[ \t]+(Base|Role:[ \t]*[A-Za-z0-9_-]+)[ \t]*$/gm;
      let m;
      while ((m = re.exec(this.promptEditor.content)) !== null) {
        out.push({ label: m[1].replace(/\s+/g, ' '), index: m.index });
      }
      return out;
    },

    gotoPromptSection(index) {
      const el = document.getElementById('prompt-editor-text');
      if (!el) return;
      el.focus();
      el.setSelectionRange(Number(index), Number(index));
      // Approximate scroll: put the section near the top of the viewport.
      // Counting newlines assumes one visual row per line, which pre-wrap
      // (voice.css) breaks for any line long enough to wrap — close enough
      // for "near the top", not exact for a wrapped line deep in a section.
      const before = this.promptEditor.content.slice(0, Number(index)).split('\n').length;
      el.scrollTop = Math.max(0, (before - 1) * 19.5); // 0.75rem * line-height 1.625
    },

    async viewPromptVersion(id) {
      id = Number(id);
      try {
        const resp = await fetch('/api/voice/prompt?version=' + encodeURIComponent(id));
        if (!resp.ok) throw new Error('version not found');
        const data = await resp.json();
        this.promptEditor.content = data.content || '';
        this.promptEditor.viewing = id;
      } catch (e) {
        this.toastError(e.message);
      }
    },

    async doSavePrompt(opts) {
      this.promptEditor.saving = true;
      try {
        const resp = await fetch('/api/voice/prompt', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(Object.assign({ content: this.promptEditor.content }, opts)),
        });
        const data = await resp.json().catch(() => ({}));
        if (!resp.ok) throw new Error(data.error || 'invalid meta-prompt');
        this.promptEditor.versions = data.versions || this.promptEditor.versions;
        this.promptEditor.viewing = data.version;
        this.toastSuccess(this.t('prompt_saved'));
      } catch (e) {
        // The message names the exact problem (a role with no section, broken
        // front matter), which is the whole point of surfacing it verbatim.
        this.toastError(e.message);
      }
      this.promptEditor.saving = false;
    },

    // Overwrites the version currently open in the editor. The original (id
    // 0) never reaches this: its button is hidden, see promptViewingIsOriginal.
    savePromptOver() {
      if (this.promptEditor.viewing === 0) return;
      return this.doSavePrompt({ version: this.promptEditor.viewing, new: false });
    },

    // Saves the edited text as a brand-new named version, asking for the
    // name — the only way an edit of the original becomes reusable, since
    // the original itself can never be overwritten.
    savePromptAsNew() {
      const name = window.prompt(this.t('prompt_name_new'));
      if (name === null) return; // cancelled
      return this.doSavePrompt({ new: true, name: name });
    },

    // Applying only moves the active pointer server-side (PromptStore.SetActive);
    // it never touches any version's content, so it is safe to call for any
    // version at any time and there is no separate "restore" action.
    async applyPromptVersion(id) {
      try {
        const resp = await fetch('/api/voice/prompt/activate', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ version: Number(id) }),
        });
        const data = await resp.json().catch(() => ({}));
        if (!resp.ok) throw new Error(data.error || 'could not apply the version');
        this.promptEditor.versions = data.versions || this.promptEditor.versions;
        this.voice.roles = data.roles || this.voice.roles;
        this.toastSuccess(this.t('prompt_applied'));
      } catch (e) {
        this.toastError(e.message);
      }
    },

    // Renames the version currently selected in the "Versiones" dropdown.
    // The original (id 0) has no button for this — see promptViewingIsOriginal
    // in the template — but bail out here too in case that ever drifts.
    renamePromptVersion(id) {
      id = Number(id);
      if (id === 0) return;
      const v = this.promptVersionById(id);
      const name = window.prompt(this.t('prompt_rename_new_name'), v ? v.name : '');
      if (name === null) return; // cancelled
      return this.doRenamePrompt(id, name);
    },

    async doRenamePrompt(id, name) {
      try {
        const resp = await fetch('/api/voice/prompt/rename', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ version: id, name }),
        });
        const data = await resp.json().catch(() => ({}));
        if (!resp.ok) throw new Error(data.error || 'could not rename the version');
        this.promptEditor.versions = data.versions || this.promptEditor.versions;
        this.toastSuccess(this.t('prompt_renamed'));
      } catch (e) {
        this.toastError(e.message);
      }
    },

    // Deletes the version currently selected in the "Versiones" dropdown,
    // after confirming. Deleting the active one falls back to the original
    // server-side (PromptStore.Delete); if that version is also the one open
    // in the editor, doDeletePrompt below swaps the textarea to whatever the
    // response says is active now, so it never keeps showing content for an
    // id that no longer exists.
    deletePromptVersion(id) {
      id = Number(id);
      if (id === 0) return;
      const v = this.promptVersionById(id);
      const label = v ? ' “' + v.name + '”' : '';
      if (!window.confirm(this.t('prompt_delete_confirm') + label + '?')) return;
      return this.doDeletePrompt(id);
    },

    async doDeletePrompt(id) {
      try {
        const resp = await fetch('/api/voice/prompt?version=' + encodeURIComponent(id), { method: 'DELETE' });
        const data = await resp.json().catch(() => ({}));
        if (!resp.ok) throw new Error(data.error || 'could not delete the version');
        this.promptEditor.versions = data.versions || this.promptEditor.versions;
        this.voice.roles = data.roles || this.voice.roles;
        if (this.promptEditor.viewing === id) {
          this.promptEditor.content = data.content || '';
          const active = this.promptActiveVersion();
          this.promptEditor.viewing = active ? active.id : 0;
        }
        this.toastSuccess(this.t('prompt_deleted'));
      } catch (e) {
        this.toastError(e.message);
      }
    },

    // --- Chat input auto-grow ---
    //
    // The textarea ships as rows="1" with a max-h-32 that never came into
    // play, so a multi-line message scrolled inside a one-line box. Growing it
    // does not replace the review panel — a rewritten prompt still needs far
    // more room than this — but it fixes typing a few lines by hand.
    autoGrow(el) {
      if (!el) return;
      el.style.height = 'auto';
      el.style.height = Math.min(el.scrollHeight, 128) + 'px';
    },

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

// --- Pane auto-resize ---
// Claude Code is a normal TUI: it wraps its own output to whatever terminal
// geometry it's given (ioctl TIOCGWINSZ), same as any other. Sessions here run
// in detached tmux windows (no real terminal client ever attaches to report a
// size), so without this they sit at tmux's default 80x24 forever and
// capture-pane hands back text already hard-wrapped at 80 cols — which the
// browser's own soft-wrap (whitespace-pre-wrap) then reflows a second time at
// whatever width the box happens to have, looking wrong in both directions
// (wasted whitespace on a wide Terminal tab, broken re-wrapping on a narrow
// grid tile). There is no Claude-side setting for this (checked against
// current docs): the fix is to keep tmux's window size in sync with how many
// characters actually fit in the pane's rendered box, exactly like a real
// terminal emulator reports on resize.

const paneResizeTimers = {};
const paneResizeLast = {};
const paneResizeObservers = {};

// stopPaneResize disconnects and forgets any ResizeObserver tracking name.
// Called before (re)watching a session's pane (its DOM node is recreated
// often — x-if on the live overlay and on every grid tile) and when a pane
// closes for good, so a stale observer never outlives its element.
function stopPaneResize(name) {
  const ro = paneResizeObservers[name];
  if (ro) { ro.disconnect(); delete paneResizeObservers[name]; }
  clearTimeout(paneResizeTimers[name]);
  delete paneResizeTimers[name];
}

// measurePaneCharCell reads the monospace cell (char width, line height)
// every pane shares — the single Terminal tab and every grid tile render
// their <pre> with the exact same font-mono/text-xs/leading-relaxed classes,
// so this is measured once (cached) via a probe appended to <body> with
// position:fixed, not inside any actual pane. That placement is load-bearing,
// not cosmetic: a first version put the probe inside the pane's own
// (scrollable) box, and reading its layout there could flip on a horizontal
// scrollbar and shrink that box's own height — which is exactly what the
// ResizeObserver watching it is watching for, so the measurement re-triggered
// its own observer. In the grid, with several tiles doing this at once, that
// tripped the browser's ResizeObserver-loop guard hard enough to crash the
// tab under Playwright. A fixed-position probe on <body> can't affect any
// pane's box, so it can't feed back into anything watching one.
let paneCharCell = null;
function measurePaneCharCell() {
  if (paneCharCell) return paneCharCell;
  const probe = document.createElement('pre');
  probe.className = 'font-mono text-xs leading-relaxed';
  Object.assign(probe.style, {
    position: 'fixed', left: '-99999px', top: '0', margin: '0', padding: '0', whiteSpace: 'pre',
  });
  probe.textContent = '0'.repeat(80) + '\n' + '0'.repeat(80);
  document.body.appendChild(probe);
  const rect = probe.getBoundingClientRect();
  document.body.removeChild(probe);
  paneCharCell = { width: rect.width / 80, lineHeight: rect.height / 2 };
  return paneCharCell;
}

// requestPaneResize computes how many columns/rows fit in a pane's box (cw x
// ch, its .tgrid-pane scroll viewport, pre for its padding) and, if that
// differs from the last size sent for this session, debounces a resize call
// to the backend (tmux resize-window). Silently a no-op while the box is
// hidden/collapsed (e.g. a minimized or non-zoomed grid tile) — nothing sane
// to measure there yet.
function requestPaneResize(name, cw, ch, pre) {
  if (cw < 20 || ch < 20) return;
  const cell = measurePaneCharCell();
  if (!cell.width || !cell.lineHeight) return;
  const style = getComputedStyle(pre);
  const padX = parseFloat(style.paddingLeft) + parseFloat(style.paddingRight);
  const padY = parseFloat(style.paddingTop) + parseFloat(style.paddingBottom);
  const cols = Math.max(1, Math.floor((cw - padX) / cell.width));
  const rows = Math.max(1, Math.floor((ch - padY) / cell.lineHeight));
  const last = paneResizeLast[name];
  if (last && last.cols === cols && last.rows === rows) return;
  clearTimeout(paneResizeTimers[name]);
  paneResizeTimers[name] = setTimeout(() => {
    paneResizeLast[name] = { cols, rows };
    fetch('/api/sessions/' + encodeURIComponent(name) + '/resize', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cols, rows }),
    }).catch(() => { paneResizeLast[name] = null; }); // best effort: retried on the next size check
  }, 300);
}

// --- ANSI rendering for the multi-session terminal grid ---
// The grid asks the backend for the pane WITH its colour codes (?color=1), so
// they have to be turned into markup here. Same discipline as highlightJSON:
// escape first, then wrap the already-escaped text in spans — never the other
// way round, or the pane content becomes an HTML injection vector.

const ANSI_OSC = /\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g;
const ANSI_SGR = /\x1b\[([0-9;]*)m/g;
// Every other CSI sequence (cursor moves, clears) carries no colour and would
// otherwise show up as literal junk; dropped, like stripANSI does server-side.
const ANSI_OTHER_CSI = /\x1b\[[0-9;?]*[ -/]*[@-~]/g;

function stripAnsiForMatch(line) {
  return line.replace(ANSI_OSC, '').replace(ANSI_SGR, '').replace(ANSI_OTHER_CSI, '');
}

const BOX_RULE_RE = /^─+$/;

// Claude Code's own input box (a full-width rule, the "❯" prompt line,
// another rule) is redundant in a grid tile: each tile already has its own
// input row below the pane for typing, so showing the native box too just
// wastes vertical space. Strip that 3-line block wherever it appears.
function stripInputBoxChrome(raw) {
  const lines = raw.split('\n');
  const out = [];
  for (let i = 0; i < lines.length; i++) {
    if (
      i + 2 < lines.length &&
      BOX_RULE_RE.test(stripAnsiForMatch(lines[i]).trim()) &&
      stripAnsiForMatch(lines[i + 1]).trim().startsWith('❯') &&
      BOX_RULE_RE.test(stripAnsiForMatch(lines[i + 2]).trim())
    ) {
      i += 2;
      continue;
    }
    out.push(lines[i]);
  }
  return out.join('\n');
}

// xterm's standard 256-colour palette, needed for 38;5;N / 48;5;N sequences:
// 0-15 the basic/bright 16, 16-231 a 6x6x6 colour cube, 232-255 a grey ramp.
function xterm256ToHex(n) {
  const basic = [
    '5c5c68', 'ff6b6b', '2ed573', 'ffa502', '6c9cff', 'e090e0', '6cd4ff', 'd8d8e0',
    '7a7a88', 'ff8787', '6ef2a0', 'ffc75f', '8fb8ff', 'f0b0f0', '9ce6ff', 'ffffff',
  ];
  if (n < 16) return basic[n];
  if (n < 232) {
    const i = n - 16;
    const levels = [0, 95, 135, 175, 215, 255];
    const r = levels[Math.floor(i / 36) % 6], g = levels[Math.floor(i / 6) % 6], b = levels[i % 6];
    return [r, g, b].map(v => v.toString(16).padStart(2, '0')).join('');
  }
  const grey = 8 + (n - 232) * 10;
  return [grey, grey, grey].map(v => v.toString(16).padStart(2, '0')).join('');
}

// ansiClass maps a basic (0-107) SGR code to a CSS class, or '' when it's not
// one we paint (unknown/unsupported attribute codes are simply ignored).
function ansiClass(code) {
  if (code === 1) return 'ansi-bold';
  if (code === 2) return 'ansi-dim';
  if (code === 3) return 'ansi-italic';
  if (code === 4) return 'ansi-underline';
  if (code >= 30 && code <= 37) return 'ansi-fg-' + (code - 30);
  if (code >= 90 && code <= 97) return 'ansi-fg-' + (code - 90 + 8);
  if (code >= 40 && code <= 47) return 'ansi-bg-' + (code - 40);
  if (code >= 100 && code <= 107) return 'ansi-bg-' + (code - 100 + 8);
  return '';
}

// ansiToHtml converts a pane capture into safe HTML with colour spans. Beyond
// the basic 16-colour codes, Claude Code's real TUI leans on 256-colour and
// truecolor SGR sequences (38;5;N / 38;2;r;g;b, and 48;… for background) —
// those take several numeric parameters together, so codes are walked by
// index rather than one at a time, unlike the basic codes above.
function ansiToHtml(raw) {
  const s = escapeHtml(raw).replace(ANSI_OSC, '');
  let out = '';
  let open = 0;
  let last = 0;
  let m;
  const closeAll = () => { while (open > 0) { out += '</span>'; open--; } };
  const openSpan = (styleOrClass, isStyle) => {
    out += isStyle ? '<span style="' + styleOrClass + '">' : '<span class="' + styleOrClass + '">';
    open++;
  };
  // Builds a safe inline colour style: every value here comes from our own
  // regex-extracted integers, never from unvalidated text, so this can't be
  // used to inject arbitrary CSS/HTML.
  const colorStyle = (prop, hex) => prop + ':#' + hex;

  ANSI_SGR.lastIndex = 0;
  while ((m = ANSI_SGR.exec(s)) !== null) {
    out += s.slice(last, m.index).replace(ANSI_OTHER_CSI, '');
    last = ANSI_SGR.lastIndex;
    // An empty parameter list means SGR 0 (reset).
    const codes = m[1] === '' ? [0] : m[1].split(';').map(Number);
    for (let i = 0; i < codes.length; i++) {
      const code = codes[i];
      if (!Number.isFinite(code)) continue;

      // Extended 256-colour / truecolor foreground (38;…) or background (48;…).
      if (code === 38 || code === 48) {
        const prop = code === 38 ? 'color' : 'background-color';
        if (codes[i + 1] === 5 && Number.isFinite(codes[i + 2])) {
          openSpan(colorStyle(prop, xterm256ToHex(codes[i + 2] & 255)), true);
          i += 2;
        } else if (codes[i + 1] === 2 && [codes[i + 2], codes[i + 3], codes[i + 4]].every(Number.isFinite)) {
          const hex = [codes[i + 2], codes[i + 3], codes[i + 4]]
            .map(v => Math.max(0, Math.min(255, v)).toString(16).padStart(2, '0')).join('');
          openSpan(colorStyle(prop, hex), true);
          i += 4;
        }
        continue;
      }

      // 0 resets everything; 22/23/24/39/49 turn one attribute off. We don't
      // track attributes individually, so any "off" code closes the open spans
      // — slightly coarse, but it can never leak an unclosed tag.
      if (code === 0 || code === 22 || code === 23 || code === 24 || code === 39 || code === 49) {
        closeAll();
        continue;
      }
      const cls = ansiClass(code);
      if (cls) openSpan(cls, false);
    }
  }
  out += s.slice(last).replace(ANSI_OTHER_CSI, '');
  closeAll();
  return out;
}
