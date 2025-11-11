import './style.css';
import './app.css';

import { GetMode } from '../wailsjs/go/main/App';
import { renderHostLobby } from './host.js';
import { renderClientUI } from './client.js';

// Initialize the app
async function initApp() {
    try {
        const currentMode = await GetMode();
        console.log('App mode:', currentMode);

        if (currentMode === 'host') {
            renderHostLobby();
        } else {
            renderClientUI();
        }
    } catch (error) {
        console.error('Failed to get mode:', error);
        // Default to client mode
        renderClientUI();
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', initApp);
