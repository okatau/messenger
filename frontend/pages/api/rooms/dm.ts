import type { NextApiRequest, NextApiResponse } from 'next'
import { API_URL } from '../../../constants'

export default async function handlerDM(req: NextApiRequest, res: NextApiResponse) {
    if (req.method !== 'POST') return res.status(405).end()

    const response = await fetch(`${API_URL}/api/v1/rooms/dm`, {
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
