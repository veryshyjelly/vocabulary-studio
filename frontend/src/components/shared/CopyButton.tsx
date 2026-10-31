import React, { useState } from "react";
import { Check, Copy } from "lucide-react";


import { Button } from "../ui";

/**
 * CopyButton copies text to the clipboard with a temporary checkmark confirmation.
 */
export const CopyButton: React.FC<{ text: string; label?: string; size?: "sm" | "md" }> = ({
  text,
  label,
  size = "md",
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      // Clipboard unavailable
    }
  };

  return (
    <Button size={size} variant="ghost" icon={copied ? Check : Copy} onClick={handleCopy} aria-live="polite">
      {copied ? "Copied" : label || "Copy"}
    </Button>
  );
};

