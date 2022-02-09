import React, {useEffect, useState} from 'react';
import './App.css';
import ws, {Channel} from "./channelsocket";

interface Object {
    name: string
    description: string
}

function useObject(name: string): [Object | null, Error | null, string] {
    const [state, setState] = useState<Object | null>(null)
    const [error, setError] = useState<Error | null>(null)
    const [key, setKey] = useState<string>("")

    useEffect(() => {
        const ch = ws.channel();
        setKey(ch.id)

        ch.onOpen((ch: Channel) => {
            ch.subscribe("object." + name)
            ch.send("object.get", name)
        })

        ch.onMessageType<Object>("object", (m: Object) => {
            setState(m)
            setError(null)
        })

        ch.onMessageType<string>("error", (m: string) => {
            setError(new Error(m))
        })

        return () => {
            ch.close()
        }
    }, [name, setState, setError, setKey])

    return [state, error, key]
}

function Object(props: { name: string }) {
    const [obj, err, key] = useObject(props.name)
    return (
        <div style={{border: "1px solid black", padding: "4px", textAlign: "left"}}>
            <h3>{props.name} ({key})</h3>
            {!obj && !err && <div>working...</div>}
            {err && <div><b>Error:</b> {err.message}</div>}
            <b>Description:</b> {obj && <pre>{obj.description}</pre>}
        </div>
    )
}

function App() {
    const [objects, setObjects] = useState(["fdasfs-12", "fdasfsa-14", "fdas-14", "foobar-14"])
    const [newObject, setNewObject] = useState("")

    return (
        <div className="App">
            {objects.map((name) => {
                return <div key={name}>
                    <Object name={name}/>
                    <button onClick={() => {
                        setObjects([...objects.filter((x) => x !== name)])
                    }}>Remove
                    </button>
                </div>
            })}
            <input value={newObject} onChange={(x) => setNewObject(x.target.value)}/>
            <button onClick={() => {
                setObjects([...objects, newObject])
                setNewObject("")
            }}>Add
            </button>
        </div>
    );
}

export default App;
