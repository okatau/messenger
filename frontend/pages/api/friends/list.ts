import type { NextApiRequest, NextApiResponse } from 'next'
import { API_URL } from '../../../constants'

export default async function handlerGetFriends(req: NextApiRequest, res: NextApiResponse) {
    if (req.method !== 'GET') {
        return res.status(405).end()
    }

    const response = await fetch(`${API_URL}/api/v1/friends/`, {
        method: 'GET',
        headers: {
            'Authorization': req.headers.authorization ?? '',
        },
    })
    const data = await response.json()
    return res.status(response.status).json(data)
}
