/*
 * Copyright (C) 2025 Red Hat, Inc.
 * SPDX-License-Identifier: LGPL-2.1-or-later
 */
import { useEffect, useState } from "react";

export const useNetworkStatus = () => {
    const [hasActiveCheckpoint, setHasActiveCheckpoint] = useState(false);

    useEffect(() => {
        const checkpointState = window.sessionStorage.getItem("cockpit_has_checkpoint");
        setHasActiveCheckpoint(checkpointState === "true");

        const handleCheckpointEvent = (event) => {
            if (event.key === "cockpit_has_checkpoint") {
                setHasActiveCheckpoint(event.newValue === "true");
            }
        };

        window.addEventListener("storage", handleCheckpointEvent);

        return () => window.removeEventListener("storage", handleCheckpointEvent);
    }, []);

    return { hasActiveCheckpoint };
};
