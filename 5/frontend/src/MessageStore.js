import { useContext, useState, createContext, useRef } from "react"

const MessageStoreContext = createContext()

export const useMessageStore = () =>
    useContext(MessageStoreContext)

export function MessageStoreProvider({children}) {
    // a BST would make it more efficient, i think
    const messageMap = useRef(new Map());
    const [messages, updateMessages] = useState([])

    const PushMessage = m => {
        messageMap.current.set(m.id, m)
        setMessagesArray()
    }

    const PushMessages = ms => {
        ms.forEach(m => messageMap.current.set(m.id, m))
        setMessagesArray()
    }

    const setMessagesArray = () => {
        const _messages = Array.from(messageMap.current.values());
        // the "correct" should be `(1 - a.timestamp === b.timestamp) * (1 - 2 * a.timestamp < b.timestamp)`
        // but i'm not sure it's actually necessary to preserve eq here
        _messages.sort((a, b) => 1 - 2 * a.timestamp > b.timestamp)
        updateMessages(_messages)
    }
    
    return <MessageStoreContext.Provider value={[
            messages, 
            PushMessage, 
            PushMessages
        ]}>
        {children}
    </MessageStoreContext.Provider>
}
