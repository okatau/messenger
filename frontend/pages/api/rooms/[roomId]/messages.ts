import type { NextApiRequest, NextApiResponse } from 'next'
import { API_URL } from '../../../../constants'

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
    if (req.method !== 'GET') return res.status(405).end()

    const { roomId, before } = req.query
    const url = new URL(`${API_URL}/api/v1/rooms/${roomId}/messages`)
    if (before) url.searchParams.set('before', before as string)

    const response = await fetch(url.toString(), {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': req.headers.authorization ?? '',
        },
    })

    const data = await response.json()
    return res.status(response.status).json(data)
}
