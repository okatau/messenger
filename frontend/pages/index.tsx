import React, { useState, useRef, useContext, useEffect, useLayoutEffect, useCallback } from 'react';

const useIsomorphicLayoutEffect = typeof window !== 'undefined' ? useLayoutEffect : useEffect;
import ChatBody from '../components/chat_body';
import { WebsocketContext } from '../modules/websocket_provider';
import { AuthContext } from '../modules/auth_provider';
import autosize from 'autosize';
import type { Message } from '../components/chat_body';

type Room = { id: string; name: string };
type SearchUser = { id: string; username: string; email: string };

const HISTORY_PAGE_SIZE = 50;

const Index = () => {
    const { user, isReady } = useContext(AuthContext);
    const { conn } = useContext(WebsocketContext);

    // Rooms
    const [rooms, setRooms] = useState<Room[]>([]);
    const [selectedRoom, setSelectedRoom] = useState<Room | null>(null);
    const [newRoomName, setNewRoomName] = useState('');
    const [showCreateRoom, setShowCreateRoom] = useState(false);

    // Messages
    const [messages, setMessages] = useState<Message[]>([]);
    const [hasMore, setHasMore] = useState(false);
    const [isLoadingHistory, setIsLoadingHistory] = useState(false);

    // Users in room
    const [roomUsers, setRoomUsers] = useState<{ username: string }[]>([]);

    // Invite
    const [showInvite, setShowInvite] = useState(false);
    const [inviteUsername, setInviteUsername] = useState('');
    const [searchResults, setSearchResults] = useState<SearchUser[]>([]);
    const [searchLoading, setSearchLoading] = useState(false);

    // Refs
    const textarea = useRef<HTMLTextAreaElement>(null);
    const bottomRef = useRef<HTMLDivElement>(null);
    const topSentinelRef = useRef<HTMLDivElement>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    const prevScrollHeightRef = useRef(0);
    const isLoadingRef = useRef(false);

    const getRooms = useCallback(async () => {
        try {
            const res = await fetch('/api/rooms', {
                headers: { Authorization: `Bearer ${user.access_token}` },
            });
            const data = await res.json();
            setRooms(data ?? []);
        } catch (e) {
            console.error(e);
        }
    }, [user.access_token]);

    useEffect(() => {
        if (isReady) getRooms();
    }, [isReady, getRooms]);

    const fetchHistory = useCallback(async (before?: string) => {
        if (isLoadingRef.current || !selectedRoom || !user?.access_token) return;
        isLoadingRef.current = true;
        setIsLoadingHistory(true);
        try {
            const url = before
                ? `/api/rooms/${selectedRoom.id}/messages?before=${encodeURIComponent(before)}`
                : `/api/rooms/${selectedRoom.id}/messages`;
            const res = await fetch(url, {
                headers: { Authorization: `Bearer ${user.access_token}` },
            });
            if (!res.ok) return;
            const data: any[] = await res.json();
            const mapped: Message[] = data.map((msg) => ({
                ...msg,
                type: user?.username === msg.username ? 'self' : 'recv',
            }));
            if (before) {
                if (containerRef.current) {
                    prevScrollHeightRef.current = containerRef.current.scrollHeight;
                }
                setMessages((prev) => [...mapped.reverse(), ...prev]);
            } else {
                setMessages(mapped.reverse());
            }
            setHasMore(data.length === HISTORY_PAGE_SIZE);
        } catch (e) {
            console.error(e);
        } finally {
            isLoadingRef.current = false;
            setIsLoadingHistory(false);
        }
    }, [selectedRoom, user]);

    // Restore scroll position after prepending old messages
    useIsomorphicLayoutEffect(() => {
        if (prevScrollHeightRef.current > 0 && containerRef.current) {
            containerRef.current.scrollTop +=
                containerRef.current.scrollHeight - prevScrollHeightRef.current;
            prevScrollHeightRef.current = 0;
        }
    }, [messages]);

    // On room select: reset state, load history, fetch active users
    useEffect(() => {
        if (!selectedRoom) return;
        setMessages([]);
        setHasMore(false);
        fetchHistory();

        fetch(`/api/rooms/${selectedRoom.id}/active`, {
            headers: { Authorization: `Bearer ${user.access_token}` },
        })
            .then((r) => r.json())
            .then((data) => setRoomUsers(data ?? []))
            .catch(console.error);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [selectedRoom?.id]);

    // IntersectionObserver: load older messages when scrolled to top
    useEffect(() => {
        if (!topSentinelRef.current || !hasMore) return;
        const observer = new IntersectionObserver(
            (entries) => {
                if (entries[0].isIntersecting && messages.length > 0) {
                    fetchHistory(messages[0].timestamp);
                }
            },
            { root: containerRef.current, threshold: 0.1 }
        );
        observer.observe(topSentinelRef.current);
        return () => observer.disconnect();
    }, [hasMore, messages, fetchHistory]);

    useEffect(() => {
        if (textarea.current) autosize(textarea.current);
    }, [selectedRoom]);

    // Auto-scroll to bottom on new incoming messages (not history prepend)
    useEffect(() => {
        if (prevScrollHeightRef.current === 0) {
            bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
        }
    }, [messages]);

    // WebSocket: realtime messages only
    useEffect(() => {
        if (!conn) return;
        conn.onmessage = (event) => {
            const m = JSON.parse(event.data);
            if (Array.isArray(m)) return; // ignore WS history batch
            if (m.message === 'joined the room') {
                setRoomUsers((prev) => [...prev, { username: m.username }]);
                return;
            }
            if (m.message === 'left the room') {
                setRoomUsers((prev) => prev.filter((u) => u.username !== m.username));
                setMessages((prev) => [...prev, m]);
                return;
            }
            m.type = user?.username === m.username ? 'self' : 'recv';
            setMessages((prev) => [...prev, m]);
        };
    }, [conn, user]);

    const sendMessage = () => {
        if (!textarea.current?.value || !conn || !selectedRoom) return;
        conn.send(JSON.stringify({ message: textarea.current.value, roomId: selectedRoom.id }));
        textarea.current.value = '';
        autosize.update(textarea.current);
    };

    const createRoom = async () => {
        if (!newRoomName.trim()) return;
        try {
            await fetch('/api/rooms', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: `Bearer ${user.access_token}`,
                },
                body: JSON.stringify({ name: newRoomName }),
            });
            setNewRoomName('');
            setShowCreateRoom(false);
            getRooms();
        } catch (e) {
            console.error(e);
        }
    };

    const leaveRoom = async () => {
        if (!selectedRoom || !user) return;
        try {
            await fetch(`/api/rooms/${selectedRoom.id}/leave`, {
                method: 'POST',
                headers: { Authorization: `Bearer ${user.access_token}` },
            });
        } catch (e) {
            console.error(e);
        }
        setSelectedRoom(null);
        setMessages([]);
        setRoomUsers([]);
        getRooms();
    };

    const handleLogout = () => {
        localStorage.removeItem('user_info');
        window.location.href = '/user/login';
    };

    useEffect(() => {
        if (!inviteUsername.trim()) {
            setSearchResults([]);
            return;
        }
        const timer = setTimeout(async () => {
            setSearchLoading(true);
            try {
                const res = await fetch(`/api/friends/search?username=${encodeURIComponent(inviteUsername)}`, {
                    headers: { Authorization: `Bearer ${user.access_token}` },
                });
                if (res.ok) setSearchResults((await res.json()) ?? []);
            } catch (e) {
                console.error(e);
            } finally {
                setSearchLoading(false);
            }
        }, 300);
        return () => clearTimeout(timer);
    }, [inviteUsername, user.access_token]);

    const inviteUser = async (userId: string) => {
        if (!selectedRoom) return;
        try {
            await fetch(`/api/rooms/${selectedRoom.id}/invite`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: `Bearer ${user.access_token}`,
                },
                body: JSON.stringify({ userId }),
            });
        } catch (e) {
            console.error(e);
        }
        setShowInvite(false);
        setInviteUsername('');
        setSearchResults([]);
    };

    if (!isReady) return null;

    return (
        <div className="flex flex-col w-full h-screen">
            {/* Header */}
            <header className="flex items-center justify-between px-6 py-3 border-b border-grey bg-white shrink-0">
                <div className="flex items-center gap-4">
                    <span className="font-bold text-blue">{user.username}</span>
                    <div className="w-px h-4 bg-grey" />
                    <button
                        className="text-sm bg-blue text-white px-3 py-1 rounded-md"
                        onClick={() => setShowCreateRoom(true)}
                    >
                        + Create Room
                    </button>
                </div>
                <button
                    className="text-sm border border-grey px-3 py-1 rounded-md text-dark-secondary hover:bg-grey transition-colors"
                    onClick={handleLogout}
                >
                    Logout
                </button>
            </header>

            {/* Three-column layout */}
            <div className="flex flex-1 overflow-hidden">
                {/* Left: chat list */}
                <aside className="w-56 border-r border-grey overflow-y-auto shrink-0 bg-white">
                    <div className="px-4 py-3 text-xs font-bold text-grey-dark uppercase tracking-wide">
                        Chats
                    </div>
                    {rooms.length === 0 && (
                        <div className="px-4 py-2 text-sm text-grey-dark">No rooms yet</div>
                    )}
                    {rooms.map((room) => (
                        <button
                            key={room.id}
                            onClick={() => setSelectedRoom(room)}
                            className={`w-full text-left px-4 py-3 text-sm transition-colors hover:bg-grey ${
                                selectedRoom?.id === room.id
                                    ? 'bg-grey font-semibold text-blue border-r-2 border-blue'
                                    : 'text-dark-secondary'
                            }`}
                        >
                            {room.name}
                        </button>
                    ))}
                </aside>

                {/* Center: chat body */}
                <main className="flex flex-col flex-1 overflow-hidden bg-white">
                    {selectedRoom ? (
                        <>
                            <div className="px-4 py-3 border-b border-grey text-sm font-semibold text-center shrink-0">
                                {selectedRoom.name}
                            </div>
                            <div ref={containerRef} className="flex-1 overflow-y-auto p-4">
                                <div ref={topSentinelRef} className="h-1" />
                                {isLoadingHistory && (
                                    <div className="text-center text-xs text-gray-400 py-2">
                                        Loading...
                                    </div>
                                )}
                                <ChatBody data={messages} />
                                <div ref={bottomRef} />
                            </div>
                            <div className="px-4 py-3 border-t border-grey shrink-0">
                                <div className="flex gap-2 items-end">
                                    <textarea
                                        ref={textarea}
                                        placeholder="Type your message..."
                                        className="flex-1 border border-grey rounded-md p-2 text-sm focus:outline-none focus:border-blue"
                                        style={{ resize: 'none', minHeight: '40px', maxHeight: '120px' }}
                                        onKeyDown={(e) => {
                                            if (e.key === 'Enter' && !e.shiftKey) {
                                                e.preventDefault();
                                                sendMessage();
                                            }
                                        }}
                                    />
                                    <button
                                        className="bg-blue text-white px-4 py-2 rounded-md text-sm shrink-0"
                                        onClick={sendMessage}
                                    >
                                        Send
                                    </button>
                                </div>
                            </div>
                        </>
                    ) : (
                        <div className="flex-1 flex items-center justify-center text-grey-dark text-sm">
                            Select a chat to start messaging
                        </div>
                    )}
                </main>

                {/* Right: room actions */}
                <aside className="w-52 border-l border-grey shrink-0 bg-white p-4">
                    {selectedRoom ? (
                        <div className="flex flex-col gap-3">
                            <div className="text-xs font-bold text-grey-dark uppercase tracking-wide">
                                Actions
                            </div>
                            <button
                                className="w-full text-sm bg-blue text-white px-3 py-2 rounded-md"
                                onClick={() => setShowInvite(true)}
                            >
                                Invite User
                            </button>
                            <button
                                className="w-full text-sm border border-grey text-dark-secondary px-3 py-2 rounded-md hover:bg-grey transition-colors"
                                onClick={leaveRoom}
                            >
                                Leave Room
                            </button>

                            {roomUsers.length > 0 && (
                                <div className="mt-4">
                                    <div className="text-xs font-bold text-grey-dark uppercase tracking-wide mb-2">
                                        Online
                                    </div>
                                    {roomUsers.map((u, i) => (
                                        <div key={i} className="text-sm py-1 text-dark-secondary">
                                            {u.username}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    ) : (
                        <div className="text-xs text-grey-dark">No room selected</div>
                    )}
                </aside>
            </div>

            {/* Create Room Modal */}
            {showCreateRoom && (
                <div
                    className="fixed inset-0 bg-black bg-opacity-30 flex items-center justify-center z-50"
                    onClick={() => { setShowCreateRoom(false); setNewRoomName(''); }}
                >
                    <div className="bg-white rounded-lg p-6 w-80 shadow-lg" onClick={(e) => e.stopPropagation()}>
                        <h3 className="font-bold mb-4">Create Room</h3>
                        <input
                            type="text"
                            className="w-full border border-grey rounded-md p-2 text-sm focus:outline-none focus:border-blue mb-4"
                            placeholder="Room name"
                            value={newRoomName}
                            onChange={(e) => setNewRoomName(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && createRoom()}
                            autoFocus
                        />
                        <div className="flex gap-2 justify-end">
                            <button
                                className="text-sm px-3 py-1 rounded-md border border-grey text-dark-secondary"
                                onClick={() => { setShowCreateRoom(false); setNewRoomName(''); }}
                            >
                                Cancel
                            </button>
                            <button
                                className="text-sm px-3 py-1 rounded-md bg-blue text-white"
                                onClick={createRoom}
                            >
                                Create
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Invite User Modal */}
            {showInvite && (
                <div
                    className="fixed inset-0 bg-black bg-opacity-30 flex items-center justify-center z-50"
                    onClick={() => { setShowInvite(false); setInviteUsername(''); setSearchResults([]); }}
                >
                    <div className="bg-white rounded-lg p-6 w-80 shadow-lg" onClick={(e) => e.stopPropagation()}>
                        <h3 className="font-bold mb-4">Invite to {selectedRoom?.name}</h3>
                        <input
                            type="text"
                            className="w-full border border-grey rounded-md p-2 text-sm focus:outline-none focus:border-blue mb-2"
                            placeholder="Search by username..."
                            value={inviteUsername}
                            onChange={(e) => setInviteUsername(e.target.value)}
                            autoFocus
                        />
                        <div className="min-h-[80px] max-h-48 overflow-y-auto mb-4">
                            {searchLoading && (
                                <div className="text-xs text-grey-dark py-2 text-center">Searching...</div>
                            )}
                            {!searchLoading && inviteUsername && searchResults.length === 0 && (
                                <div className="text-xs text-grey-dark py-2 text-center">No users found</div>
                            )}
                            {searchResults.map((u) => (
                                <div key={u.id} className="flex items-center justify-between py-2 border-b border-grey last:border-0">
                                    <span className="text-sm text-dark-secondary">{u.username}</span>
                                    <button
                                        className="text-xs bg-blue text-white px-2 py-1 rounded-md"
                                        onClick={() => inviteUser(u.id)}
                                    >
                                        Invite
                                    </button>
                                </div>
                            ))}
                        </div>
                        <div className="flex justify-end">
                            <button
                                className="text-sm px-3 py-1 rounded-md border border-grey text-dark-secondary"
                                onClick={() => { setShowInvite(false); setInviteUsername(''); setSearchResults([]); }}
                            >
                                Cancel
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default Index;
