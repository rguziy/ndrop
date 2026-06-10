const els = {
   apiKey: document.querySelector("#api-key"),
   rememberKey: document.querySelector("#remember-key"),
   toggleKey: document.querySelector("#toggle-key"),
   statusPill: document.querySelector("#status-pill"),
   pullTab: document.querySelector("#pull-tab"),
   pushTab: document.querySelector("#push-tab"),
   pullView: document.querySelector("#pull-view"),
   pushView: document.querySelector("#push-view"),
   pullButton: document.querySelector("#pull-button"),
   copyButton: document.querySelector("#copy-button"),
   downloadButton: document.querySelector("#download-button"),
   pullMeta: document.querySelector("#pull-meta"),
   pullText: document.querySelector("#pull-text"),
   pushText: document.querySelector("#push-text"),
   pushTextButton: document.querySelector("#push-text-button"),
   pushFile: document.querySelector("#push-file"),
   pushFileButton: document.querySelector("#push-file-button"),
   message: document.querySelector("#message"),
};
const authForm = document.querySelector("#auth-form");
const dropzone = document.getElementById('dropzone');
const dropzonePrompt = document.querySelector('.dropzone-prompt');

const storageKey = "ndrop.apiKey";
let lastDownload = null;

function setMessage(text, kind = "") {
   els.message.textContent = text;
   els.message.className = `message ${kind}`.trim();
}

function apiKey() {
   return els.apiKey.value.trim();
}

function requireAPIKey() {
   const key = apiKey();
   if (!key) {
      throw new Error("Enter API key first.");
   }
   return key;
}

function updateAuthState() {
   const hasKey = apiKey().length > 0;
   els.statusPill.innerHTML = hasKey ?
      `<svg class="icon"><use href="#icon-check"></use></svg><span>Ready</span>` :
      `<svg class="icon"><use href="#icon-lock"></use></svg><span>Locked</span>`;
   els.statusPill.classList.toggle("ok", hasKey);
   if (els.rememberKey.checked && hasKey) {
      localStorage.setItem(storageKey, apiKey());
   } else if (!els.rememberKey.checked) {
      localStorage.removeItem(storageKey);
   }
}

function showTab(name) {
   const isPull = name === "pull";
   els.pullTab.classList.toggle("active", isPull);
   els.pushTab.classList.toggle("active", !isPull);
   els.pullView.classList.toggle("active", isPull);
   els.pushView.classList.toggle("active", !isPull);
}

function bytesToBase64(bytes) {
   let binary = "";
   const chunkSize = 0x8000;
   for (let i = 0; i < bytes.length; i += chunkSize) {
      binary += String.fromCharCode(...bytes.subarray(i, i + chunkSize));
   }
   return btoa(binary);
}

function base64ToBytes(value) {
   const binary = atob(value);
   const bytes = new Uint8Array(binary.length);
   for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
   }
   return bytes;
}

async function deriveKey(keyText) {
   const source = await crypto.subtle.importKey(
      "raw",
      new TextEncoder().encode(keyText),
      "HKDF",
      false,
      ["deriveKey"],
   );
   return crypto.subtle.deriveKey({
         name: "HKDF",
         hash: "SHA-256",
         salt: new Uint8Array(),
         info: new TextEncoder().encode("ndrop-encrypt"),
      },
      source, {
         name: "AES-GCM",
         length: 256
      },
      false,
      ["encrypt", "decrypt"],
   );
}

async function encryptPayload(keyText, plaintext) {
   const key = await deriveKey(keyText);
   const nonce = crypto.getRandomValues(new Uint8Array(12));
   const ciphertext = await crypto.subtle.encrypt({
      name: "AES-GCM",
      iv: nonce
   }, key, plaintext);
   return {
      data: bytesToBase64(new Uint8Array(ciphertext)),
      nonce: bytesToBase64(nonce),
   };
}

async function decryptPayload(keyText, data, nonce) {
   const key = await deriveKey(keyText);
   const plaintext = await crypto.subtle.decrypt({
         name: "AES-GCM",
         iv: base64ToBytes(nonce)
      },
      key,
      base64ToBytes(data),
   );
   return new Uint8Array(plaintext);
}

function deviceName() {
   const platform = navigator.platform || "web";
   return `web-${platform}`.slice(0, 80);
}

async function pushPayload({
   type,
   name = "",
   mime,
   bytes
}) {
   const key = requireAPIKey();
   const encrypted = await encryptPayload(key, bytes);
   const resp = await fetch("/push", {
      method: "POST",
      headers: {
         "Content-Type": "application/json",
         Authorization: `Bearer ${key}`,
      },
      body: JSON.stringify({
         device: deviceName(),
         type,
         name,
         mime,
         data: encrypted.data,
         nonce: encrypted.nonce,
      }),
   });
   if (!resp.ok) {
      throw new Error(await responseError(resp, "push failed"));
   }
}

async function responseError(resp, fallback) {
   const body = await resp.text();
   return body.trim() || `${fallback}: HTTP ${resp.status}`;
}

function setBusy(button, busy) {
   button.disabled = busy;
   button.dataset.originalText ||= button.textContent;
   button.textContent = busy ? "Working..." : button.dataset.originalText;
}

async function pullLatest() {
   const key = requireAPIKey();
   setBusy(els.pullButton, true);
   setMessage("Pulling latest...");
   els.copyButton.disabled = true;
   els.downloadButton.disabled = true;
   els.pullText.value = "";
   lastDownload = null;

   try {
      const resp = await fetch("/pull", {
         headers: {
            Authorization: `Bearer ${key}`
         },
      });
      if (resp.status === 204) {
         els.pullMeta.textContent = "Buffer is empty or expired.";
         setMessage("Nothing to pull.");
         return;
      }
      if (!resp.ok) {
         throw new Error(await responseError(resp, "pull failed"));
      }

      const entry = await resp.json();
      const plaintext = await decryptPayload(key, entry.data, entry.nonce);
      const type = entry.type || "text";
      const name = entry.name || "ndrop-download";
      const mime = entry.mime || "application/octet-stream";
      els.pullMeta.textContent = `${type} from ${entry.device || "unknown device"} · ${name} · ${plaintext.byteLength} bytes`;

      if (type === "text") {
         els.pullText.value = new TextDecoder().decode(plaintext);
         els.copyButton.disabled = false;
      } else {
         const downloadName = type === "folder" && !name.toLowerCase().endsWith(".zip") ? `${name}.zip` : name;
         lastDownload = {
            bytes: plaintext,
            mime,
            name: downloadName
         };
         els.pullText.value = type === "folder" ?
            "Folder transfer loaded as a zip file. Use Download to save it." :
            "File transfer loaded. Use Download to save it.";
         els.downloadButton.disabled = false;
      }
      setMessage("Pulled and decrypted locally.", "ok");
   } finally {
      setBusy(els.pullButton, false);
   }
}

async function pushText() {
   const text = els.pushText.value;
   if (!text) {
      throw new Error("Text is empty.");
   }
   setBusy(els.pushTextButton, true);
   try {
      await pushPayload({
         type: "text",
         mime: "text/plain",
         bytes: new TextEncoder().encode(text),
      });
      setMessage("Text pushed.", "ok");
      els.pushText.value = "";
   } finally {
      setBusy(els.pushTextButton, false);
   }
}

async function pushFile() {
   const file = els.pushFile.files && els.pushFile.files[0];
   if (!file) {
      throw new Error("Choose a file first.");
   }
   setBusy(els.pushFileButton, true);
   try {
      await pushPayload({
         type: "file",
         name: file.name,
         mime: file.type || "application/octet-stream",
         bytes: new Uint8Array(await file.arrayBuffer()),
      });
      setMessage(`File pushed: ${file.name}`, "ok");
      resetDropzone();
   } finally {
      setBusy(els.pushFileButton, false);
   }
}

async function copyPulledText() {
   await navigator.clipboard.writeText(els.pullText.value);
   setMessage("Copied to clipboard.", "ok");
}

function downloadPulled() {
   if (!lastDownload) {
      return;
   }
   const blob = new Blob([lastDownload.bytes], {
      type: lastDownload.mime
   });
   const url = URL.createObjectURL(blob);
   const a = document.createElement("a");
   a.href = url;
   a.download = lastDownload.name;
   document.body.appendChild(a);
   a.click();
   a.remove();
   URL.revokeObjectURL(url);
}

function bindAsync(el, event, fn) {
   el.addEventListener(event, () => {
      fn().catch((err) => setMessage(err.message || String(err), "error"));
   });
}

function updateDropzoneFileInfo() {
   const file = els.pushFile.files && els.pushFile.files[0];
   if (file) {
      const fileSize = (file.size / 1024).toFixed(1);
      dropzonePrompt.innerHTML = `
      <svg class="icon upload-cloud" aria-hidden="true" style="color: var(--ok);"><use href="#icon-check"></use></svg>
      <span class="file-info-text">Selected: <strong>${file.name}</strong> (${fileSize} KB)</span>
      <button type="button" class="clear-file-btn" id="clear-file" title="Remove file">Cancel</button>
    `;
      dropzone.classList.add('has-file');

      document.getElementById('clear-file').addEventListener('click', (e) => {
         e.stopPropagation();
         e.preventDefault();
         resetDropzone();
      });
   } else {
      resetDropzone();
   }
}

function resetDropzone() {
   els.pushFile.value = '';
   dropzonePrompt.innerHTML = `
    <svg class="icon upload-cloud" aria-hidden="true"><use href="#icon-upload"></use></svg>
    <span><strong>Click to upload</strong> or drag and drop</span>
  `;
   dropzone.classList.remove('has-file');
}

function init() {
   if (!window.crypto || !crypto.subtle) {
      setMessage("WebCrypto is not available. Use HTTPS or localhost.", "error");
   }

   const saved = localStorage.getItem(storageKey);
   if (saved) {
      els.apiKey.value = saved;
      els.rememberKey.checked = true;
   }
   updateAuthState();

   if (dropzone) {
      dropzone.addEventListener('dragover', (e) => {
         e.preventDefault();
         dropzone.classList.add('dragover');
      });

      ['dragleave', 'drop'].forEach(eventName => {
         dropzone.addEventListener(eventName, () => {
            dropzone.classList.remove('dragover');
         });
      });
   }
   els.apiKey.addEventListener("input", updateAuthState);
   els.rememberKey.addEventListener("change", updateAuthState);
   els.toggleKey.addEventListener("click", () => {
      const show = els.apiKey.type === "password";
      els.apiKey.type = show ? "text" : "password";
      els.toggleKey.querySelector('use').setAttribute('href', show ? '#icon-eye-off' : '#icon-eye');
   });
   authForm.addEventListener("submit", (event) => event.preventDefault());
   els.pullTab.addEventListener("click", () => showTab("pull"));
   els.pushTab.addEventListener("click", () => showTab("push"));
   els.pushFile.addEventListener("change", updateDropzoneFileInfo);
   bindAsync(els.pullButton, "click", pullLatest);
   bindAsync(els.copyButton, "click", copyPulledText);
   els.downloadButton.addEventListener("click", downloadPulled);
   bindAsync(els.pushTextButton, "click", pushText);
   bindAsync(els.pushFileButton, "click", pushFile);
}

init();
