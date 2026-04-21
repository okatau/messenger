import { useState, createContext, useEffect } from 'react'
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
}>({
  authenticated: false,
  setAuthenticated: () => {},
  user: { username: '', id: '', refresh_token:'', access_token:'' },
  setUser: () => {},
  isReady: false,
})

const AuthContextProvider = ({ children }: { children: React.ReactNode }) => {
  const [authenticated, setAuthenticated] = useState(false)
  const [user, setUser] = useState<UserInfo>({  username: '', id: '', refresh_token:'', access_token:'' })
  const [isReady, setIsReady] = useState(false)

  const router = useRouter()

  useEffect(() => {
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

      fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: storedUser.refresh_token }),
      })
        .then((res) => {
          if (!res.ok) throw new Error('refresh failed')
          return res.json()
        })
        .then((data) => {
          const refreshedUser: UserInfo = {
            username: data.username,
            id: data.user_id,
            access_token: data.access_token,
            refresh_token: data.refresh_token,
          }
          localStorage.setItem('user_info', JSON.stringify(refreshedUser))
          setUser(refreshedUser)
          setAuthenticated(true)
          setIsReady(true)
        })
        .catch(() => {
          localStorage.removeItem('user_info')
          router.push('/user/login')
        })
    }
  }, [authenticated])

  return (
    <AuthContext.Provider
      value={{
        authenticated: authenticated,
        setAuthenticated: setAuthenticated,
        user: user,
        setUser: setUser,
        isReady: isReady,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export default AuthContextProvider
