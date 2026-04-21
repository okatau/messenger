import React from 'react';

export type Message = {
    message: string;
    userId: string;
    username: string;
    roomId: string;
    timestamp: string;
    type: 'recv' | 'self';
};

const ChatBody = ({ data }: { data: Array<Message> }) => {
    return (
        <>
            {data.map((message: Message, index: number) => {
                const messageDate = new Date(message.timestamp).toLocaleTimeString([], {
                    hour: '2-digit',
                    minute: '2-digit',
                });

                if (message.type === 'self') {
                    return (
                        <div className='flex flex-col mt-2 w-full text-right justify-end' key={index}>
                            <div className='text-sm'>{message.username}</div>
                            <div>
                                <div className='bg-blue text-white px-4 py-1 rounded-md inline-block mt-1'>
                                    {message.message}
                                </div>
                                <div className='text-xs text-gray-500 mt-1'>{messageDate}</div>
                            </div>
                        </div>
                    );
                } else {
                    return (
                        <div className='mt-2' key={index}>
                            <div className='text-sm'>{message.username}</div>
                            <div>
                                <div className='bg-grey text-dark-secondary px-4 py-1 rounded-md inline-block mt-1'>
                                    {message.message}
                                </div>
                                <div className='text-xs text-gray-500 mt-1'>{messageDate}</div>
                            </div>
                        </div>
                    );
                }
            })}
        </>
    );
};

export default ChatBody;
