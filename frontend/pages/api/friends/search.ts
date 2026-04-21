import type { NextApiRequest, NextApiResponse } from 'next'
import { API_URL } from '../../../constants'

export default async function handlerSearchUsers(req: NextApiRequest, res: NextApiResponse) {
    if (req.method !== 'GET') {
        return res.status(405).end()
    }

    const { username, cursor } = req.query
    const params = new URLSearchParams()
    if (username) params.set('username', String(username))
    if (cursor) params.set('cursor', String(cursor))

    const response = await fetch(`${API_URL}/api/v1/friends/search?${params}`, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': req.headers.authorization ?? '',
        },
    })
    const data = await response.json()
    return res.status(response.status).json(data)
}
