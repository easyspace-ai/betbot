import { useEffect, useState } from "react";

function AppContent() {
  const [serverStatus, setServerStatus] = useState<string>("Checking...");
  const [version, setVersion] = useState<string>("");

  useEffect(() => {
    window.desktopAPI.appInfo.then((info) => {
      setVersion(info.version);
    });

    fetch("http://localhost:8080/health")
      .then((res) => res.json())
      .then((data) => setServerStatus(data.status))
      .catch(() => setServerStatus("offline"));
  }, []);

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        fontFamily: "system-ui, sans-serif",
        padding: "20px",
      }}
    >
      <h1 style={{ fontSize: "2.5rem", marginBottom: "20px" }}>🤖 Betbot</h1>
      <p style={{ fontSize: "1.2rem", color: "#666" }}>Simple Desktop Demo</p>

      <div
        style={{
          marginTop: "40px",
          padding: "20px",
          background: "#f5f5f5",
          borderRadius: "8px",
          minWidth: "300px",
        }}
      >
        <h2 style={{ fontSize: "1rem", marginBottom: "15px" }}>Status</h2>
        <p>
          <strong>App Version:</strong> {version || "loading..."}
        </p>
        <p>
          <strong>Server:</strong> {serverStatus}
        </p>
      </div>

      <button
        onClick={() => {
          window.desktopAPI.openExternal("https://betbot.ai");
        }}
        style={{
          marginTop: "20px",
          padding: "10px 20px",
          background: "#0070f3",
          color: "white",
          border: "none",
          borderRadius: "6px",
          cursor: "pointer",
          fontSize: "1rem",
        }}
      >
        Open External Link
      </button>
    </div>
  );
}

export default function App() {
  return <AppContent />;
}