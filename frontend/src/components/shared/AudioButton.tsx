import React, { useEffect, useRef, useState } from "react";
import { Volume2, Loader2, VolumeX } from "lucide-react";

import { getBridge } from "../../lib/bridge";
import { cx } from "../ui";

/**
 * AudioButton plays a pronunciation MP3 URL or resolves the audio URL for a word on demand.
 */
export const AudioButton: React.FC<{
  audioUrl?: string;
  word?: string;
  label?: string;
  ariaLabel?: string;
  size?: "sm" | "md";
  variant?: "secondary" | "ghost" | "soft";
  className?: string;
}> = ({ audioUrl, word, label, ariaLabel, size = "md", variant = "secondary", className = "" }) => {
  const [playing, setPlaying] = useState(false);
  const [failed, setFailed] = useState(false);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  useEffect(
    () => () => {
      audioRef.current?.pause();
    },
    []
  );

  const handlePlay = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (playing) return;
    setFailed(false);
    setPlaying(true);

    try {
      let urlToPlay = audioUrl;
      if (!urlToPlay && word) {
        urlToPlay = await getBridge().dictionary.ResolveAudioURL(word);
      }
      if (!urlToPlay) {
        setFailed(true);
        setPlaying(false);
        return;
      }
      audioRef.current?.pause();
      const audio = new Audio(urlToPlay);
      audioRef.current = audio;
      audio.onended = () => setPlaying(false);
      audio.onerror = () => {
        setFailed(true);
        setPlaying(false);
      };
      await audio.play();
    } catch {
      setFailed(true);
      setPlaying(false);
    }
  };

  const Icon = playing ? Loader2 : failed ? VolumeX : Volume2;
  const accessible = `Play pronunciation${word ? ` for ${word}` : ""}`;

  return (
    <button
      type="button"
      onClick={handlePlay}
      title={failed ? "Audio unavailable" : accessible}
      aria-label={`${ariaLabel || (label ? `${label}${word ? ` (${word})` : ""}` : accessible)}${failed ? "; audio unavailable, try again" : ""}`}
      aria-busy={playing || undefined}
      className={cx(
        "inline-flex items-center justify-center gap-1.5 shrink-0 rounded-control font-nav font-medium",
        "transition-[background-color,border-color,color,transform] duration-150 ease-out-soft active:translate-y-px",
        size === "sm" ? (label ? "h-7 px-2.5 text-meta" : "h-7 w-7") : label ? "h-8.5 px-3 text-ui" : "h-8.5 w-8.5",
        failed
          ? "border border-warning-line bg-warning-soft text-warning-ink"
          : playing
          ? "border border-accent-line bg-accent-soft text-accent-ink"
          : variant === "ghost"
          ? "text-ink-muted hover:text-accent-ink hover:bg-sunken"
          : variant === "soft"
          ? "bg-accent-soft text-accent-ink hover:bg-accent-soft-hover"
          : "border border-line bg-surface text-ink hover:border-line-strong hover:bg-sunken",
        className
      )}
    >
      <Icon className={cx("w-3.5 h-3.5", playing && "animate-spin", !failed && !playing && "text-accent-ink")} aria-hidden />
      {label && <span>{label}</span>}
    </button>
  );
};

