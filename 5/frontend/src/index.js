import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import App from './App';

import { MessageStoreProvider } from './MessageStore';
import { UserStoreProvider } from './UserStore';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <MessageStoreProvider>
    <UserStoreProvider>
      <App />
    </UserStoreProvider>
    </MessageStoreProvider>
  </React.StrictMode>
);
