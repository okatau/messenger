const apiBase = process.env.NEXT_PUBLIC_API_URL || 'http://localhost'

export const API_URL = process.env.API_INTERNAL_URL || apiBase

export const WS_URL = apiBase
  .replace(/^https:/, 'wss:')
  .replace(/^http:/, 'ws:') + '/api/v1/rooms/wss'
