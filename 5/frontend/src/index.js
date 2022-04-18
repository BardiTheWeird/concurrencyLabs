import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import App from './App';

import { AdapterMoment } from '@mui/x-date-pickers/AdapterMoment'
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider'

import { MessageStoreProvider } from './MessageStore';
import { UserStoreProvider } from './UserStore';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <LocalizationProvider dateAdapter={AdapterMoment} >
      <MessageStoreProvider>
      <UserStoreProvider>
        <App />
      </UserStoreProvider>
      </MessageStoreProvider>
    </LocalizationProvider>
  </React.StrictMode>
);
