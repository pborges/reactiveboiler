import {nanoid} from "nanoid";

export interface Message {
    client: string
    type: string
    channel?: string
    body: any
}

const protocolTypes: string[] = ["open", "close"]

class Socket {
    private readonly log: Log
    private readonly addr: string
    private ws: WebSocket
    private channels: Channel[] = []
    private queue: Message[] = []
    readonly id: string

    constructor(addr: string) {
        this.id = nanoid()
        this.log = new Log(this.id)
        this.addr = addr
        this.ws = this.setup()
    }

    private setup(): WebSocket {
        const ws = new WebSocket(this.addr)
        ws.onopen = () => {
            this.log.info("open")
            ws.send(this.id)
            this.drain()
            this.channels.forEach((ch: Channel) => {
                ch.send("open")
                ch.onOpenCallbacks.forEach((cb) => cb(ch))
            })
        }
        ws.onclose = () => {
            this.log.info("close")
            setTimeout(() => {
                this.log.info("reconnecting....")
                this.ws = this.setup()
            }, 5000)
        }
        ws.onerror = (e: Event) => {
            this.log.error(e)
        }
        ws.onmessage = (e: MessageEvent) => {
            const msg = JSON.parse(e.data) as Message
            if (msg.channel) {
                this.channels.filter((ch) => ch.id === msg.channel)
                    .forEach((ch) => {
                        ch.log.recv(msg)
                        ch.onMessageCallbacks.forEach((cb) => {
                            cb(msg)
                        })
                    })
            } else {
                this.log.recv(msg)
            }
        }

        return ws
    }

    private drain() {
        if (this.ws.readyState === WebSocket.OPEN) {
            this.queue.forEach((m: Message) => {
                const msg = JSON.stringify(m)
                if (m.channel) {
                    this.channels.filter((ch) => ch.id === m.channel)
                        .forEach((ch) => {
                            if (protocolTypes.filter((x) => x === m.type).length > 0) {
                                ch.log.info(m.type)
                            } else {
                                ch.log.send(m)
                            }
                        })
                } else {
                    this.log.send(m)
                }
                this.ws.send(msg)
            })
            this.queue = []
        }
    }

    channel(): Channel {
        const ch = new Channel(this);
        this.channels.push(ch)
        return ch
    }

    send(m: Message) {
        this.queue.push({...m, client: this.id})
        this.drain()
    }

    isOpen(): boolean {
        return this.ws.readyState === WebSocket.OPEN
    }

    close(channel: string) {
        const ch = this.channels.filter((x) => x.id === channel).pop()
        ch && ch.send("close")
        this.channels = this.channels.filter((x) => x.id !== channel)
    }
}

export class Channel {
    private readonly socket: Socket
    readonly log: Log
    readonly id: string
    readonly onOpenCallbacks: ((ch: Channel) => void)[] = []
    readonly onMessageCallbacks: ((m: Message) => void)[] = []

    constructor(socket: Socket) {
        this.id = nanoid()
        this.socket = socket
        this.log = new Log(socket.id, this.id)
    }

    send(type: string, m: any = undefined) {
        this.socket.send({...m, channel: this.id, type: type, body: m})
    }

    subscribe(topic: string): Channel {
        this.send("subscribe", topic)
        return this
    }

    onMessage(cb: (m: Message) => void): Channel {
        this.onMessageCallbacks.push(cb)
        return this
    }

    onMessageType<T>(type: string, cb: (x: T) => void): Channel {
        return this.onMessage((m: Message) => {
            if (m.type === type) {
                cb(m.body as T)
            }
        })
    }

    onOpen(cb: (ch: Channel) => void): Channel {
        this.onOpenCallbacks.push(cb)
        if (this.socket.isOpen()) {
            cb(this)
        }
        return this
    }

    close() {
        this.socket.close(this.id)
    }
}


class Log {
    private readonly client: string
    private readonly channel?: string

    constructor(client: string, channel: string | undefined = undefined) {
        this.client = client
        this.channel = channel
    }

    send(msg: Message) {
        this.writeto(console.log, "<<", "#CAEEBE", msg.type, msg.body)
    }

    recv(msg: Message) {
        this.writeto(console.log, ">>", "#D3EEFF", msg.type, msg.body)
    }

    info(...msg: any[]) {
        this.writeto(console.log, "**", "#EEEBE5", ...msg)
    }

    debug(...msg: any[]) {
        this.writeto(console.debug, "??", "#EEEBE5", ...msg)
    }

    error(...msg: any[]) {
        this.writeto(console.error, "**", "#FEBCC8", ...msg)
    }

    private writeto(fn: (...m: any[]) => void, symbol: string, color: string, ...msg: any[]) {
        const identifier = this.channel ? "[" + this.client + " -> " + this.channel + "]" : "[" + this.client + "]"
        fn("%c[WS " + symbol + "]" + identifier, "color:" + color, ...msg)
    }
}


const ws = new Socket("ws://localhost:8080/ws")
export default ws