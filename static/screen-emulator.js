export default class ScreenEmulator {
    constructor(width, height, scale = 1) {
        this.width = width;
        this.height = height;
        this.scale = scale;
        this.buffer = new Uint8Array(width * height * 4); // RGBA 格納用(Go 側から一括転送される)

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

        // 等倍描画用のオフスクリーンキャンバスと ImageData は 1 回だけ生成して再利用する
        this.offscreen = document.createElement("canvas");
        this.offscreen.width = width;
        this.offscreen.height = height;
        this.offCtx = this.offscreen.getContext("2d");
        this.imageData = this.offCtx.createImageData(width, height);

        // DOM に追加
        this.container.appendChild(this.canvas);
    }

    size() {
        return { x: this.width, y: this.height };
    }

    setPixel(x, y, r, g, b, a) {
        if (x < 0 || x >= this.width || y < 0 || y >= this.height) return;
        const i = (y * this.width + x) * 4;
        this.buffer[i] = r;
        this.buffer[i + 1] = g;
        this.buffer[i + 2] = b;
        this.buffer[i + 3] = a;
    }

    display() {
        this.imageData.data.set(this.buffer);
        this.offCtx.putImageData(this.imageData, 0, 0);

        // 小さいキャンバスに描画した内容を拡大して表示
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        this.ctx.drawImage(this.offscreen, 0, 0, this.canvas.width, this.canvas.height);
    }
}
