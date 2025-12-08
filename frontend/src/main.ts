import './style.css';
import { setTransport } from '@wailsio/runtime';
import { Greet, GetMode, ListAllUsers } from '../bindings/GOproject/app.js';

// Wails v3 alpha backend expects query-based runtime calls; provide custom transport.
setTransport({
  call: async (objectID, method, windowName, args) => {
    const url = new URL('/wails/runtime', window.location.origin);
    url.searchParams.set('object', String(objectID));
    url.searchParams.set('method', String(method));
    if (args !== null && args !== undefined) {
      url.searchParams.set('args', JSON.stringify(args));
    }

    const headers: Record<string, string> = {};
    if (windowName) {
      headers['x-wails-window-name'] = windowName;
    }

    const response = await fetch(url.toString(), { method: 'GET', headers });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    const ct = response.headers.get('Content-Type') || '';
    if (ct.includes('application/json')) {
      return response.json();
    }
    return response.text();
  },
});

const appEl = document.querySelector('#app');

if (!appEl) {
  throw new Error('Root element #app not found');
}

appEl.innerHTML = `
  <div class="card">
    <h1>GOproject</h1>
    <p class="small">Wails v3 bridge rebuilt. Try calling Go methods below.</p>
  </div>
  <div class="card">
    <label for="name">Greet</label>
    <input id="name" placeholder="Your name" value="Wails" />
    <button id="btn-greet">Send to Go</button>
    <pre id="greet-output">(waiting)</pre>
  </div>
  <div class="grid">
    <div class="card">
      <label>Get current mode</label>
      <button id="btn-mode">GetMode()</button>
      <pre id="mode-output">(waiting)</pre>
    </div>
    <div class="card">
      <label>List all users</label>
      <button id="btn-users">ListAllUsers()</button>
      <pre id="users-output">(waiting)</pre>
    </div>
  </div>
`;

const greetOutput = document.querySelector('#greet-output') as HTMLElement;
const modeOutput = document.querySelector('#mode-output') as HTMLElement;
const usersOutput = document.querySelector('#users-output') as HTMLElement;
const nameInput = document.querySelector('#name') as HTMLInputElement;

async function handleGreet() {
  try {
    const name = nameInput.value.trim() || 'Wails';
    const msg = await Greet(name);
    greetOutput.textContent = msg;
  } catch (err) {
    greetOutput.textContent = `Error: ${String(err)}`;
  }
}

async function handleMode() {
  try {
    const mode = await GetMode();
    modeOutput.textContent = mode;
  } catch (err) {
    modeOutput.textContent = `Error: ${String(err)}`;
  }
}

async function handleUsers() {
  try {
    const users = await ListAllUsers();
    usersOutput.textContent = JSON.stringify(users, null, 2);
  } catch (err) {
    usersOutput.textContent = `Error: ${String(err)}`;
  }
}

(document.querySelector('#btn-greet') as HTMLButtonElement).addEventListener('click', handleGreet);
(document.querySelector('#btn-mode') as HTMLButtonElement).addEventListener('click', handleMode);
(document.querySelector('#btn-users') as HTMLButtonElement).addEventListener('click', handleUsers);

handleGreet();
handleMode();
handleUsers();
