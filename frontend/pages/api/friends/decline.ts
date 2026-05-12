import type { NextApiRequest, NextApiResponse } from 'next'
import { API_URL } from '../../../constants'

export default async function handlerDeclineFriend(req: NextApiRequest, res: NextApiResponse) {
    if (req.method !== 'POST') {
        return res.status(405).end()
    }

    const response = await fetch(`${API_URL}/api/v1/friends/decline`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': req.headers.authorization ?? '',
        },
        body: JSON.stringify(req.body),
    })
    const data = await response.json()
    return res.status(response.status).json(data)
}
