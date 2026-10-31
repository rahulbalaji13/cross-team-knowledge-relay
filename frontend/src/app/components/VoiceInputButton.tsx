"use client";

import { useEffect, useRef, useState } from "react";

type SpeechAlternative = { transcript: string };
type SpeechResult = { length: number; [index: number]: SpeechAlternative };
type SpeechResultList = { length: number; [index: number]: SpeechResult };
type SpeechResultEvent = { results: SpeechResultList };
type SpeechErrorEvent = { error: string };

type SpeechRecognitionLike = {
  lang: string;
  continuous: boolean;
  interimResults: boolean;
  onresult: ((event: SpeechResultEvent) => void) | null;
  onerror: ((event: SpeechErrorEvent) => void) | null;
  onend: (() => void) | null;
  start: () => void;
  stop: () => void;
  abort: () => void;
};

type SpeechRecognitionConstructor = new () => SpeechRecognitionLike;
type SpeechWindow = Window & {
  SpeechRecognition?: SpeechRecognitionConstructor;
  webkitSpeechRecognition?: SpeechRecognitionConstructor;
};

export default function VoiceInputButton({
  fieldLabel,
  onTranscript,
}: {
  fieldLabel: string;
  onTranscript: (transcript: string) => void;
}) {
  const [isListening, setIsListening] = useState(false);
  const [feedback, setFeedback] = useState("");
  const recognitionRef = useRef<SpeechRecognitionLike | null>(null);

  useEffect(() => () => {
    recognitionRef.current?.abort();
    recognitionRef.current = null;
  }, []);

  function toggleListening() {
    if (isListening) {
      recognitionRef.current?.stop();
      setIsListening(false);
      setFeedback("Voice input stopped.");
      return;
    }

    if (!window.isSecureContext) {
      setFeedback("Voice input needs a secure connection (HTTPS).");
      return;
    }

    const speechWindow = window as SpeechWindow;
    const Recognition = speechWindow.SpeechRecognition || speechWindow.webkitSpeechRecognition;
    if (!Recognition) {
      setFeedback("Voice input is not supported in this browser. You can still type here.");
      return;
    }

    const recognition = new Recognition();
    recognition.lang = navigator.language || "en-US";
    recognition.continuous = false;
    recognition.interimResults = false;
    recognition.onresult = (event) => {
      const result = event.results[event.results.length - 1];
      const transcript = result?.[0]?.transcript?.trim();
      if (transcript) {
        onTranscript(transcript);
        setFeedback("Added. Review the text before posting.");
      } else {
        setFeedback("No speech was recognized. Try again or type instead.");
      }
    };
    recognition.onerror = (event) => {
      setIsListening(false);
      recognitionRef.current = null;
      if (event.error === "not-allowed" || event.error === "service-not-allowed") {
        setFeedback("Microphone access was blocked. Allow it in your browser settings and try again.");
      } else if (event.error === "no-speech") {
        setFeedback("No speech was detected. Try again or type instead.");
      } else if (event.error === "audio-capture") {
        setFeedback("No microphone was found. Connect one or type instead.");
      } else {
        setFeedback("Voice input could not start. Try again or type instead.");
      }
    };
    recognition.onend = () => {
      setIsListening(false);
      recognitionRef.current = null;
    };

    recognitionRef.current = recognition;
    setFeedback("Listening… speak now.");
    try {
      recognition.start();
      setIsListening(true);
    } catch {
      recognitionRef.current = null;
      setIsListening(false);
      setFeedback("Voice input could not start. Try again or type instead.");
    }
  }

  return (
    <div className="shrink-0">
      <button
        type="button"
        onClick={toggleListening}
        aria-label={isListening ? "Stop voice input for " + fieldLabel : "Dictate " + fieldLabel}
        aria-pressed={isListening}
        title={isListening ? "Stop voice input" : "Dictate " + fieldLabel}
        className={"inline-flex min-h-11 items-center justify-center gap-1.5 rounded-xl border px-3 text-xs font-semibold transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 " + (isListening ? "border-indigo-300 bg-indigo-50 text-indigo-700" : "border-slate-200 bg-white text-slate-600 hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-700")}
      >
        {isListening ? <span className="size-2 animate-pulse rounded-full bg-indigo-600" aria-hidden="true" /> : <MicrophoneIcon />}
        <span>{isListening ? "Stop" : "Speak"}</span>
      </button>
      {feedback ? <p role={feedback.startsWith("Microphone") || feedback.startsWith("Voice input is not") || feedback.startsWith("Voice input needs") ? "alert" : "status"} className="mt-1.5 max-w-56 text-xs leading-4 text-slate-500">{feedback}</p> : null}
    </div>
  );
}

function MicrophoneIcon() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-4">
    <rect x="9" y="3" width="6" height="11" rx="3" stroke="currentColor" strokeWidth="1.7" />
    <path d="M5.5 11.5a6.5 6.5 0 0 0 13 0M12 18v3m-3 0h6" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
  </svg>;
}