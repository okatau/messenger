import { useState, createContext, useEffect, useRef } from 'react'
import { useRouter } from 'next/router'

export type UserInfo = {
  username: string
  id: string
  refresh_token: string
  access_token: string
}

export const AuthContext = createContext<{
  authenticated: boolean
  setAuthenticated: (auth: boolean) => void
  user: UserInfo
  setUser: (user: UserInfo) => void
  isReady: boolean
  setIsReady: (ready: boolean) => void
}>({
  authenticated: false,
  setAuthenticated: () => {},
  user: { username: '', id: '', refresh_token:'', access_token:'' },
  setUser: () => {},
  isReady: false,
  setIsReady: () => {},
})

const AuthContextProvider = ({ children }: { children: React.ReactNode }) => {
  const [authenticated, setAuthenticated] = useState(false)
  const [user, setUser] = useState<UserInfo>({  username: '', id: '', refresh_token:'', access_token:'' })
  const [isReady, setIsReady] = useState(false)

  const router = useRouter()
  const hasRun = useRef(false)
  const refreshTokenRef = useRef('')

  const doRefresh = async (refreshToken: string): Promise<UserInfo> => {
    const res = await fetch('/api/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) throw new Error('refresh failed')
    const data = await res.json()
    return {
      username: data.username,
      id: data.user_id,
      access_token: data.access_token,
      refresh_token: data.refresh_token,
    }
  }

  useEffect(() => {
    if (hasRun.current) return
    hasRun.current = true

    const userInfo = localStorage.getItem('user_info')

    if (!userInfo) {
      if (window.location.pathname != '/user/register') {
        router.push('/user/login')
        return
      } else {
        router.push('/user/register')
        return
      }
    } else {
      const storedUser: UserInfo = JSON.parse(userInfo)

      doRefresh(storedUser.refresh_token)
        .then((refreshedUser) => {
          localStorage.setItem('user_info', JSON.stringify(refreshedUser))
          refreshTokenRef.current = refreshedUser.refresh_token
          setUser(refreshedUser)
          setAuthenticated(true)
          setIsReady(true)
        })
        .catch(() => {
          localStorage.removeItem('user_info')
          router.push('/user/login')
        })
    }
  }, [])

  // Keep refreshTokenRef in sync when user is set externally (e.g. after login)
  useEffect(() => {
    if (user.refresh_token) {
      refreshTokenRef.current = user.refresh_token
    }
  }, [user.refresh_token])

  // Proactively refresh the access token every 14 minutes (TTL is 15m)
  useEffect(() => {
    if (!isReady) return

    const interval = setInterval(async () => {
      try {
        const refreshedUser = await doRefresh(refreshTokenRef.current)
        localStorage.setItem('user_info', JSON.stringify(refreshedUser))
        refreshTokenRef.current = refreshedUser.refresh_token
        setUser(refreshedUser)
      } catch {
        localStorage.removeItem('user_info')
        router.push('/user/login')
      }
    }, 14 * 60 * 1000)

    return () => clearInterval(interval)
  }, [isReady])

  return (
    <AuthContext.Provider
      value={{
        authenticated: authenticated,
        setAuthenticated: setAuthenticated,
        user: user,
        setUser: setUser,
        isReady: isReady,
        setIsReady: setIsReady,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export default AuthContextProvider
