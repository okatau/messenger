import type { NextApiRequest, NextApiResponse } from 'next'
import { API_URL } from '../../../../constants'

export default async function handlerInviteUser(req: NextApiRequest, res: NextApiResponse) {
    if (req.method !== 'POST') return res.status(405).end()

    const { roomId } = req.query

    const response = await fetch(`${API_URL}/api/v1/rooms/${roomId}/invite`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': req.headers.authorization ?? '',
        },
        body: JSON.stringify(req.body),
    })

    return res.status(response.status).end()
}
