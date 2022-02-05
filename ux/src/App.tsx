import React, {useEffect, useState} from 'react';
import './App.css';
import ws, {Channel} from "./channelsocket";

interface Jira {
    issue: string
    description: string
}

function useJira(issue: string): [Jira | null, Error | null, string] {
    const [state, setState] = useState<Jira | null>(null)
    const [error, setError] = useState<Error | null>(null)
    const [key, setKey] = useState<string>("")

    useEffect(() => {
        const ch = ws.channel();
        setKey(ch.id)

        ch.onOpen((ch: Channel) => {
            ch.subscribe("jira." + issue)
            ch.send("jira.get", issue)
        })

        ch.onMessageType<Jira>("jira", (m: Jira) => {
            setState(m)
            setError(null)
        })

        ch.onMessageType<string>("error", (m: string) => {
            setError(new Error(m))
        })

        return () => {
            ch.close()
        }
    }, [issue, setState, setError, setKey])

    return [state, error, key]
}

function Jira(props: { issue: string }) {
    const [jira, err, key] = useJira(props.issue)
    return (
        <div style={{border: "1px solid black", padding: "4px", textAlign: "left"}}>
            <h3>{props.issue} ({key})</h3>
            {!jira && !err && <div>working...</div>}
            {err && <div><b>Error:</b> {err.message}</div>}
            <b>Description:</b> {jira && <pre>{jira.description}</pre>}
        </div>
    )
}

function App() {
    const [jiras, setJiras] = useState(["devx-12", "devx-14", "quark-14", "foobar-14"])
    const [newJira, setNewJira] = useState("")
    return (
        <div className="App">
            {jiras.map((j) => {
                return <div key={j}>
                    <Jira issue={j}/>
                    <button onClick={() => {
                        setJiras([...jiras.filter((x) => x !== j)])
                    }}>Remove
                    </button>
                </div>
            })}
            <input value={newJira} onChange={(x) => setNewJira(x.target.value)}/>
            <button onClick={() => {
                setJiras([...jiras, newJira])
                setNewJira("")
            }}>Add
            </button>
        </div>
    );
}

export default App;
