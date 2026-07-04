import ScreenEmulator from "./screen-emulator.js";

const screenContainer = document.querySelector('.screen-container');
let screen = new ScreenEmulator(128, 64, 5);
screenContainer.appendChild(screen.canvas);

let keysPressed = {}; // 押されているキーを記録

async function loadWASM() {
    const go = new Go();
    const wasmModule = await WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject);
    go.run(wasmModule.instance);

    requestAnimationFrame(processKeys); // キー入力処理を開始
}

// **キーが押されたときに記録**
document.addEventListener("keydown", (event) => {
    keysPressed[event.key] = true;
});

// **キーが離されたときに削除**
document.addEventListener("keyup", (event) => {
    delete keysPressed[event.key];
});

// **キーを処理するループ**
function processKeys() {
    if (window.wasmKeyEvent) {
        for (const key in keysPressed) {
            if (keysPressed[key]) {
                window.wasmKeyEvent(key); // すべての押されているキーを送信
            }
        }
    }
    requestAnimationFrame(processKeys); // 次のフレームもチェック
}

screen.canvas.addEventListener("touchstart", (event) => {
    event.preventDefault();

    if (window.wasmKeyEvent) {
        window.wasmKeyEvent("0");
    }
});

// (Displayed only on touch devices (forced display and for testing by appending
// `?touch` to the URL).)
function setupTouchControls() {
    const forced = new URLSearchParams(location.search).has("touch");
    const isTouch = window.matchMedia("(pointer: coarse)").matches || "ontouchstart" in window;
    if (!forced && !isTouch) return;
    document.body.classList.add("touch");

    document.querySelectorAll(".pad-btn[data-key]").forEach((btn) => {
        const key = btn.dataset.key;
        const press = (event) => {
            event.preventDefault();
            // 指がボタンの外へ滑っても pointerup を受け取れるように capture する
            if (btn.setPointerCapture && event.pointerId !== undefined) {
                btn.setPointerCapture(event.pointerId);
            }
            keysPressed[key] = true;
            btn.classList.add("pressed");
        };
        const release = (event) => {
            event.preventDefault();
            delete keysPressed[key];
            btn.classList.remove("pressed");
        };
        btn.addEventListener("pointerdown", press);
        btn.addEventListener("pointerup", release);
        btn.addEventListener("pointercancel", release);
        btn.addEventListener("contextmenu", (event) => event.preventDefault()); // 長押しメニューを抑止
    });
}
setupTouchControls();

// Go 側から js.CopyBytesToJS で 1 フレーム分を一括転送するためのバッファ
window.screenBuffer = screen.buffer;

// 旧 API との互換用（現在の Go 側からは呼ばれない）
window.setPixel = (x, y, r, g, b, a) => {
    screen.setPixel(x, y, r, g, b, a);
};

// URL に ?perf を付けると FPS を console に出力する
const perfEnabled = new URLSearchParams(location.search).has("perf");
let perfFrames = 0;
let perfLast = performance.now();

window.display = () => {
    screen.display();

    if (perfEnabled) {
        perfFrames++;
        const now = performance.now();
        if (now - perfLast >= 1000) {
            console.log(`fps: ${(perfFrames * 1000 / (now - perfLast)).toFixed(1)}`);
            perfFrames = 0;
            perfLast = now;
        }
    }
};

window.clearScreen = () => {
    screen.buffer.fill(0);
};

loadWASM();
