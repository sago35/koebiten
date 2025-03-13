import ScreenEmulator from "./screen-emulator.js";

let screen = new ScreenEmulator(128, 64, 5);
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

window.setPixel = (x, y, r, g, b, a) => {
    screen.setPixel(x, y, { r, g, b, a });
};

window.display = () => {
    screen.display();
};

window.clearScreen = () => {
    screen.buffer.fill(0);
};

loadWASM();
