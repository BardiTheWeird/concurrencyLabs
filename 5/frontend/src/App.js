import './App.css'
import { useEffect, useRef, useState } from 'react'

import TextField from '@mui/material/TextField'
import { AdapterMoment } from '@mui/x-date-pickers/AdapterMoment'
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'

import { useMessageStore } from './MessageStore'
import { useUserStore } from './UserStore'

const moment = require('moment')

function App() {
  const [username, updateUsername] = useState()
  const usernamePrompt = useRef("")
  const [loggedIn, updateLoggedIn] = useState(false)
  const [messages, pushMessage, pushMessages] = useMessageStore()
  const [users, setUser, setUsers] = useUserStore()

  const [receiversType, updateReceiversType] = useState('all')
  const selectedReceivers = useRef(new Set())

  const [shouldSchedule, updateShouldSchedule] = useState(false)
  const [scheduledDateTime, updateScheduledDateTime] = useState(moment())

  const [curTime, updateCurTime] = useState(moment())
  useEffect(() => {
    const id = setInterval(() => updateCurTime(moment()))
    return () => clearInterval(id)
  }, [])

  useEffect(() => {
    if (shouldSchedule) {
      const tNow = moment()
      updateScheduledDateTime(tNow.add(1, 'minutes'))
    }
  }, [shouldSchedule])

  const messagesEndRef = useRef(null)
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth" })
    }
  }, [messages])

  const messageToSend = useRef(null)
  const ws = useRef(null)

  // const [ws, updateWs] = useState()
  useEffect(() => {
    const socket = new WebSocket("ws://127.0.0.1:8080")
    socket.onopen = () => {
      ws.current = socket

      ws.current.onmessage = msg => {
        const { kind, data } = JSON.parse(msg.data)
        console.log(`kind: ${kind}, data: ${JSON.stringify(data)}`)
        switch (kind) {
          case 'login_status':
            switch (data) {
              case 'ok':
                updateLoggedIn(true)
                updateUsername(usernamePrompt.current)
                console.log('logged in as', usernamePrompt.current)
                break
              case 'bad_username':
                alert('bad username')
                break
              case 'already_logged_in':
                alert('already logged in')
                break
              default:
                break
            }
            break
          
          case 'users': 
            setUsers(data)
          break
          case 'user_logged_in':
            console.log(data, 'logged in')
            setUser({username: data, online: true})
            break
          case 'user_logged_out':
            console.log(data, 'logged out')
            setUser({username: data, online: false})
            break

          case 'message_history':
            pushMessages(data.map(m => {return {
              ...m,
              timestamp: new Date(m.timestamp),
            }}))
          break
          case 'new_message':
            pushMessage({
              ...data,
              timestamp: new Date(data.timestamp),
            })
            break
          case 'send_success':
            if (messageToSend.current) {
              const { id, timestamp } = data
              const newMsg = {
                id: id,
                timestamp: new Date(timestamp),
                sender: username,
                body: messageToSend.current,
                receivers: Array.from(selectedReceivers.current.keys())
              }
              pushMessage(newMsg)

              messageToSend.current = null
            }
            break
          case 'send_fail':
            if (messageToSend.current) {
              messageToSend.current = null
              console.log('failed seding a message:', data)
            }
            break
          default:
            break
        }
      }
    }

    return () => {
      if (ws.current) {
        ws.current.close()
      }
    }
  }, [])

  const msgPrompt = useRef("")
  
  const ifConnected = f => {
    return (...args) => {
      if (!ws) {
        console.log('not connected')
        return
      }
      f(...args)
    }
  }

  const LogIn = ifConnected(() => {
    console.log('trying to log in as', usernamePrompt.current)
    ws.current.send(JSON.stringify({
      kind: "log_in",
      data: usernamePrompt.current,
    }))
  })

  const SendMessage = ifConnected(() => {
    if (!loggedIn) {
      console.log('not logged in')
      return
    }
    
    const msg = {body: msgPrompt.current};
    if (receiversType === "selected") {
      msg.receivers = Array.from(selectedReceivers.current.keys())
    }
    if (shouldSchedule) {
      msg.timestamp = scheduledDateTime.toDate()
    }

    console.log('sending message', msg)

    messageToSend.current = msgPrompt.current
    ws.current.send(JSON.stringify({
      kind: "send_message",
      data: msg,
    }))
  })

return (
<div className="App">
  <h1>Welcome to hell, I guess</h1>

  { !username &&
    <>
    <div className='loginForm'>
      <input className='textField usernameField' 
        type="text" 
        placeholder='username'
        onChange={e => usernamePrompt.current = e.target.value}/>
      <button className='submitButton loginButton' onClick={() => {
        LogIn()
      }}>Log in</button>
    </div>
    </>
    ||<div className='interface'>
        <div className='messagesUsers'>
          <div className='messagesBlock'>
            <h2>Logged in as {username}</h2>
            <div className='messages'>
            {
              messages.map(v => <div className={`message ${v.sender === username && "ownMessage" || ""}`}>
                <div className='messageMetadata'>
                  {v.sender} at {`${v.timestamp.getDay()}.${v.timestamp.getMonth()}.${v.timestamp.getFullYear()}, ${v.timestamp.getHours()}:${v.timestamp.getMinutes()}:${v.timestamp.getSeconds()}`}
                </div>
                <div className='messageBody'>{v.body}</div>
              </div>)
            }
              <div ref={messagesEndRef}/>
            </div>
          </div>
        <div className='usersBlock'>
          <h2>Users</h2>
          <div className='users'>
          {
            users.map(({username, online}) => <div className='userStatus'>
              <input type="checkbox" className='userCheckbox' 
                onChange={e => {
                  if (e.target.checked) {
                    selectedReceivers.current.add(username)
                  }
                  else {
                    selectedReceivers.current.delete(username)
                  }
                }}
              />
              <div className={`username ${!online && "userOffline" || ""}`}>
                {username}
              </div>
            </div>)
          }
          </div>
        </div>
      </div>
      <div className='messageInput'>
        <div className='messageInputAndSendButton'>
          <input className='textField messagInputField' type="textarea" wrap='hard'
            onChange={e => msgPrompt.current = e.target.value}/>
          <button className='submitButton sendMessageButton' onClick={() => {
            SendMessage()
          }}>Send</button>
        </div>
        <div className='sendControls'>
          <div className='radio receiverRadio' 
            onChange={(e) => updateReceiversType(e.target.value)}>
            <input type="radio" name='receiver' 
              checked={receiversType==="all"} 
              value="all" /> All
            <input type="radio" name='receiver' 
              checked={receiversType==="selected"} 
              value="selected" /> Selected
          </div>
          <div className='schedulePicker'>
            <div>
              <input type="checkbox" 
                checked={shouldSchedule} 
                onChange={e => updateShouldSchedule(e.target.checked) } />
            </div>
            <LocalizationProvider dateAdapter={AdapterMoment} >
              <DateTimePicker
                renderInput={(props) => <TextField variant="standard" size="small" {...props} />}
                value={scheduledDateTime}
                onChange={(newValue) => {
                  updateScheduledDateTime(newValue);
                }}
                disabled={!shouldSchedule}
                minDateTime={shouldSchedule ? curTime : moment('1970-01-01')}
              />
            </LocalizationProvider>
            
            {/* <input type="datetime-local" id="meeting-time"
              name="meeting-time" value="2018-06-12T19:30"
              min="2018-06-07T00:00" max="2018-06-14T00:00"></input> */}
        </div>
        </div>
      </div>
    </div>
  }
</div>
)
}

export default App
