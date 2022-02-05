import {nanoid} from "nanoid";
import {useEffect, useRef, useState} from "react";

export interface Message {
    client: string
    stream?: string
    body: any
}

class StreamSocket {
    readonly id: string
    private ws: WebSocket
    private streams: Map<string, StreamHandler> = new Map<string, StreamHandler>()
    private queue: Message[] = []
    private readonly addr: string

    constructor(addr: string) {
        this.id = nanoid()
        this.addr = addr;
        this.ws = this.setup()
    }

    private setup(): WebSocket {
        this.ws = new WebSocket(this.addr)
        this.ws.onmessage = (msg: MessageEvent) => {
            const m: Message = JSON.parse(msg.data)
            const streamInfo = m.stream ? "[" + m.stream + "]" : ""
            console.log("%c[WS:" + this.id + " >>]" + streamInfo, "color:#D3EEFF", m.body)
            if (m.stream) {
                const strm = this.streams.get(m.stream)
                if (strm) {
                    strm.onMessage(m)
                }
            }
        }
        this.ws.onerror = (evt: Event) => {
            console.error("%c[WS:" + this.id + " !!]", "color:#FEBCC8", evt)
            this.streams.forEach((strm: StreamHandler) => strm.onError(evt))
        }
        this.ws.onopen = () => {
            console.log("%c[WS:" + this.id + " **] open", "color:#EEEBE5")
            this.ws.send(this.id)
            this.drain()
            this.streams.forEach((strm: StreamHandler) => strm.onOpen())
        }
        this.ws.onclose = () => {
            console.log("%c[WS:" + this.id + " **] close", "color:#EEEBE5")
            this.streams.forEach((strm: StreamHandler) => strm.onClose())
            setTimeout(() => {
                console.log("%c[WS:" + this.id + " **] reconnect...", "color:#EEEBE5")
                this.setup()
            }, 5000)
        }
        return this.ws
    }

    stream(...onOpen: TypedMessageWithCallback[]): Stream {
        const handler = new StreamHandler(this)
        handler.onopen = onOpen
        this.streams.set(handler.id, handler)
        return new Stream(handler)
    }

    send(msg: any, stream: string | undefined = undefined) {
        const m: Message = {
            client: this.id,
            stream: stream,
            body: msg,
        }
        this.queue.push(m)
        this.drain()
    }

    private drain() {
        if (this.ws.readyState === WebSocket.OPEN) {
            this.queue.forEach((m: Message) => {
                const streamInfo = m.stream ? "[" + m.stream + "]" : ""
                console.log("%c[WS:" + this.id + " <<]" + streamInfo, "color:#CAEEBE", JSON.parse(JSON.stringify(m.body)))
                this.ws.send(JSON.stringify(m))
            })
            this.queue = []
        }
    }
}

class StreamHandler {
    private ss: StreamSocket
    readonly id: string
    onopen: TypedMessageWithCallback[] = []
    onmsg: ((msg: any) => void)[] = []

    constructor(ss: StreamSocket) {
        this.ss = ss
        this.id = nanoid()
    }

    onMessage(msg: Message) {
        this.onmsg.forEach((cb) => {
            cb(msg)
        })
        this.onopen.forEach((m: TypedMessageWithCallback) => {
            m.onMessage(msg)
        })
    }

    onOpen() {
        this.onopen.forEach((m) => {
            this.send(m)
        })
    }

    onClose() {
    }

    onError(evt: Event) {
    }

    send(msg: any) {
        this.ss.send(msg, this.id)
    }
}

export class Stream {
    private handler: StreamHandler

    constructor(handler: StreamHandler) {
        this.handler = handler
    }

    send(msg: TypedMessage): Stream {
        this.handler.send(msg)
        return this
    }

    id(): string {
        return this.handler.id
    }
}

const ws = new StreamSocket("ws://localhost:8080/ws")

export class Subscribe<T> {
    readonly _type: string
    readonly topic: string
    cb: (m: T) => void

    onMessage(m: Message) {
        this.cb(m.body as T)
    }

    constructor(topic: string, cb: (m: any) => void) {
        this._type = "subscribe"
        this.topic = topic
        this.cb = cb
    }
}

export interface TypedMessage {
    _type: string
}

export interface TypedMessageWithCallback {
    _type: string
    onMessage: (m: Message) => void
}

export function useStream(...onOpen: TypedMessageWithCallback[]): Stream {
    const [stream, setStream] = useState<Stream>(ws.stream())
    const open = useRef(onOpen)
    useEffect(() => {
        setStream(ws.stream(...open.current))
    }, [open, setStream])
    return stream
}
