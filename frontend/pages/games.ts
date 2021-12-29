import { SECOND } from "../constants";
import "../styles/games.css";

class Ball {
  x: number;
  y: number;
  radius: number;

  constructor(x: number, y: number, r: number) {
    this.x = x;
    this.y = y;
    this.radius = r;
  }

  draw(ctx: CanvasRenderingContext2D) {
    ctx.beginPath();
    ctx.arc(this.x, this.y, this.radius, 0, 2 * Math.PI);
    ctx.lineWidth = 1;
    ctx.fillStyle = "#000000";
    ctx.stroke();
  }
}

interface Bounds {
  width: number;
  height: number;
}

const main = () => {
  let canvas = document.getElementById("canvas") as HTMLCanvasElement | null;
  if (canvas == null) {
    throw new Error("could not get canvas");
  }

  const ctx = canvas.getContext("2d");
  if (ctx == null) {
    throw new Error("could not get rendering context");
  }
  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;

  let ball = new Ball(canvas.width / 2, canvas.height / 2, 10);
  const draw = () => {
    console.log("draw");
    ball.draw(ctx);
  };
  setInterval(draw, SECOND);
};

main();
