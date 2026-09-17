"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Eraser, Trash2 } from "lucide-react";

/**
 * Tiny in-room whiteboard.
 *
 * Renders an HTML canvas and lets participants draw freehand lines.
 * Strokes are kept in component state only — there is no network
 * sync yet.  A real deployment would pipe each stroke segment over
 * the SFU data channel or a dedicated Yjs document.
 */
export interface WhiteboardProps {
  width?: number;
  height?: number;
  className?: string;
}

type Point = { x: number; y: number };

export function Whiteboard({
  width = 1280,
  height = 720,
  className,
}: WhiteboardProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const drawingRef = useRef(false);
  const lastRef = useRef<Point | null>(null);
  const [hasContent, setHasContent] = useState(false);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.fillStyle = "#ffffff";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
  }, []);

  const getPoint = (
    event: React.MouseEvent<HTMLCanvasElement>,
  ): Point | null => {
    const canvas = canvasRef.current;
    if (!canvas) return null;
    const rect = canvas.getBoundingClientRect();
    const scaleX = canvas.width / rect.width;
    const scaleY = canvas.height / rect.height;
    return {
      x: (event.clientX - rect.left) * scaleX,
      y: (event.clientY - rect.top) * scaleY,
    };
  };

  const handleStart = (event: React.MouseEvent<HTMLCanvasElement>) => {
    const point = getPoint(event);
    if (!point) return;
    drawingRef.current = true;
    lastRef.current = point;
    setHasContent(true);
  };

  const handleMove = (event: React.MouseEvent<HTMLCanvasElement>) => {
    if (!drawingRef.current) return;
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext("2d");
    const point = getPoint(event);
    if (!ctx || !point || !lastRef.current) return;
    ctx.strokeStyle = "#0f172a";
    ctx.lineWidth = 3;
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    ctx.beginPath();
    ctx.moveTo(lastRef.current.x, lastRef.current.y);
    ctx.lineTo(point.x, point.y);
    ctx.stroke();
    lastRef.current = point;
  };

  const handleEnd = () => {
    drawingRef.current = false;
    lastRef.current = null;
  };

  const clearCanvas = () => {
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext("2d");
    if (!canvas || !ctx) return;
    ctx.fillStyle = "#ffffff";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    setHasContent(false);
  };

  return (
    <div
      data-testid="whiteboard"
      className={
        "flex flex-col gap-2 rounded-xl bg-slate-800 p-2 " + (className ?? "")
      }
    >
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold text-white">Whiteboard</h4>
        <Button
          variant="ghost"
          size="icon"
          onClick={clearCanvas}
          aria-label="Xóa bảng"
          disabled={!hasContent}
        >
          <Trash2 className="h-4 w-4 text-white" />
        </Button>
      </div>
      <canvas
        ref={canvasRef}
        width={width}
        height={height}
        className="block w-full cursor-crosshair rounded-lg bg-white"
        onMouseDown={handleStart}
        onMouseMove={handleMove}
        onMouseUp={handleEnd}
        onMouseLeave={handleEnd}
      />
      <p className="flex items-center gap-1 text-xs text-slate-400">
        <Eraser className="h-3 w-3" />
        Vẽ tự do bằng chuột (chưa đồng bộ qua mạng)
      </p>
    </div>
  );
}

export default Whiteboard;
