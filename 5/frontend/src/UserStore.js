import { createContext, useContext, useRef, useState } from "react"

const UserStoreContex = createContext()

export const useUserStore = () =>
    useContext(UserStoreContex)

export function UserStoreProvider({children}) {
    const usersMap = useRef(new Map())
    const [users, updateUsers] = useState([])

    const SetUser = ({username, online}) => {
        usersMap.current.set(username, online)
        setUsersArray()
    }

    const SetUsers = _users => {
        _users.forEach(({username, online}) => {
            usersMap.current.set(username, online)
            setUsersArray()
        })
    }

    const setUsersArray = () => {
        // returns [[username, online],...]
        const _users = Array.from(usersMap.current.entries())
        _users.sort(([a], [b]) => a.localeCompare(b))
        _users.sort(([, a], [, b]) => b - a)
        updateUsers(_users.map(([username, online]) => {return {
            username: username,
            online: online,
        }}))
    }

    return <UserStoreContex.Provider value={[
            users, 
            SetUser, 
            SetUsers
        ]}>
        {children}
    </UserStoreContex.Provider>
}
