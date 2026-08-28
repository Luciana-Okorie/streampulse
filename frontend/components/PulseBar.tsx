"use client";

import { useEffect, useRef } from "react";

/**
 * The signature element of the dashboard: a live oscilloscope-style trace
 * across the top of the page. Its amplitude and speed track events/sec, so
 * the whole page has a visible "pulse" — the literal thesis of StreamPulse —
 * before you read a single number.
 */
export function PulseBar({ eventsPerSecond }: { eventsPerSecond: number }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const phase = useRef(0);
  const history = useRef<number[]>(new Array(64).fill(0));

  useEffect(() => {
    history.current.push(eventsPerSecond);
    history.current.shift();
  }, [eventsPerSecond]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    let raf: number;
    const render = () => {
      const { width, height } = canvas.getBoundingClientRect();
      canvas.width = width * devicePixelRatio;
      canvas.height = height * devicePixelRatio;
      ctx.scale(devicePixelRatio, devicePixelRatio);
      ctx.clearRect(0, 0, width, height);

      const max = Math.max(...history.current, 10);
      const midY = height / 2;
      const step = width / (history.current.length - 1);

      ctx.beginPath();
      ctx.strokeStyle = "#FF8A3D";
      ctx.lineWidth = 2;
      ctx.shadowColor = "rgba(255,138,61,0.6)";
      ctx.shadowBlur = 6;

      history.current.forEach((v, i) => {
        const amplitude = (v / max) * (height * 0.4);
        const wobble = Math.sin(phase.current + i * 0.5) * (amplitude * 0.15);
        const y = midY - amplitude - wobble;
        const x = i * step;
        if (i === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      });
      ctx.stroke();

      phase.current += 0.08;
      raf = requestAnimationFrame(render);
    };

    render();
    return () => cancelAnimationFrame(raf);
  }, []);

  return (
    <div className="w-full h-16 border-b border-line bg-surface/50">
      <canvas ref={canvasRef} className="w-full h-full block" />
    </div>
  );
}
