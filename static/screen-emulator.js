export default class ScreenEmulator {
    constructor(width, height, scale = 1) {
        this.width = width;
        this.height = height;
        this.scale = scale;
        this.buffer = new Uint32Array(width * height); // RGBA 格納用

        // 外枠用の div を作成
        this.container = document.createElement("div");
        this.container.style.display = "inline-block";
        this.container.style.border = "4px solid black"; // 外枠
        this.container.style.padding = "10px"; // キャンバスとの間隔
        this.container.style.backgroundColor = "#222"; // 背景色（黒に近い）
        this.container.style.boxShadow = "0 0 10px rgba(0, 0, 0, 0.5)"; // 影を追加

        // Canvas 作成
        this.canvas = document.createElement("canvas");
        this.ctx = this.canvas.getContext("2d");

        this.canvas.width = width * scale;
        this.canvas.height = height * scale;
        this.ctx.imageSmoothingEnabled = false; // ピクセルを綺麗に保つ
        this.canvas.style.display = "block";

        // DOM に追加
        this.container.appendChild(this.canvas);
        document.body.appendChild(this.container);
    }

    size() {
        return { x: this.width, y: this.height };
    }

    setPixel(x, y, { r, g, b, a }) {
        if (x < 0 || x >= this.width || y < 0 || y >= this.height) return;
        const index = y * this.width + x;
        this.buffer[index] = (a << 24) | (r << 16) | (g << 8) | b;
    }

    display() {
        const imageData = this.ctx.createImageData(this.width, this.height);
        const data = new Uint32Array(imageData.data.buffer);

        for (let i = 0; i < this.buffer.length; i++) {
            data[i] = this.buffer[i];
        }

        // 小さいキャンバスに描画し、拡大して表示
        const tempCanvas = document.createElement("canvas");
        tempCanvas.width = this.width;
        tempCanvas.height = this.height;
        const tempCtx = tempCanvas.getContext("2d");
        tempCtx.putImageData(imageData, 0, 0);

        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        this.ctx.drawImage(tempCanvas, 0, 0, this.canvas.width, this.canvas.height);
    }
}
