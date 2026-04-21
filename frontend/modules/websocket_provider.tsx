import React, { useState, createContext, useContext, useEffect, useRef } from 'react'
import { AuthContext } from './auth_provider'
import { WEBSOCKET_URL } from '../constants'

type Conn = WebSocket | null

export const WebsocketContext = createContext<{
  conn: Conn
}>({
  conn: null,
})

const WebSocketProvider = ({ children }: { children: React.ReactNode }) => {
  const [conn, setConn] = useState<Conn>(null)
  const { user } = useContext(AuthContext)
  const connRef = useRef<Conn>(null)

  useEffect(() => {
    if (!user.access_token) {
      if (connRef.current) {
        connRef.current.close()
        connRef.current = null
        setConn(null)
      }
      return
    }

    if (connRef.current) return

    const ws = new WebSocket(`${WEBSOCKET_URL}/wss`)

    ws.onopen = () => {
      ws.send(JSON.stringify({ token: user.access_token }))
      connRef.current = ws
      setConn(ws)
    }

    ws.onclose = () => {
      connRef.current = null
      setConn(null)
    }

    ws.onerror = (e) => {
      console.error('WebSocket error', e)
      connRef.current = null
      setConn(null)
    }
  }, [user.access_token])

  return (
    <WebsocketContext.Provider value={{ conn }}>
      {children}
    </WebsocketContext.Provider>
  )
}

export default WebSocketProvider
