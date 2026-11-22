/*
 * Copyright (C) 2025 Red Hat, Inc.
 * SPDX-License-Identifier: LGPL-2.1-or-later
 */
import cockpit from "cockpit";

import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { Alert } from "@patternfly/react-core/dist/esm/components/Alert/index.js";
import { Button } from "@patternfly/react-core/dist/esm/components/Button/index.js";

import { useMaybeBackdrop } from "../../hooks/CockpitIntegration.jsx";

import { useNetworkStatus } from "./useNetworkStatus.js";

import "./NetworkConfiguration.scss";

const _ = cockpit.gettext;
const COCKPIT_NETWORK_IFRAME_SRC = "/cockpit/@localhost/network/index.html";
const GHCR_CHECK_URL = "https://ghcr.io/v2/";

export const CockpitNetworkIframe = ({
    className,
    iframeId,
    iframeName,
    onCritFail,
}) => {
    const iframeRef = useRef(null);

    useLayoutEffect(() => {
        window.sessionStorage.setItem("cockpit_anaconda", "{}");
    }, []);

    useEffect(() => {
        const el = iframeRef.current;
        if (!el) {
            return undefined;
        }

        let removeErrorListener = () => {};

        const attach = () => {
            removeErrorListener();
            const win = el.contentWindow;
            if (!win) {
                return;
            }
            const handler = exception => {
                onCritFail({ context: _("Network plugin failed") })({
                    message: exception.error.message,
                    stack: exception.error.stack,
                });
            };
            win.addEventListener("error", handler);
            removeErrorListener = () => win.removeEventListener("error", handler);
        };

        el.addEventListener("load", attach);
        attach();

        return () => {
            el.removeEventListener("load", attach);
            removeErrorListener();
        };
    }, [onCritFail]);

    return (
        <iframe
          ref={iframeRef}
          src={COCKPIT_NETWORK_IFRAME_SRC}
          name={iframeName}
          id={iframeId}
          className={className} />
    );
};

export const NetworkConfiguration = ({
    onCritFail,
    setIsFormDisabled,
    setIsFormValid,
}) => {
    const { hasActiveCheckpoint } = useNetworkStatus();
    const [registryStatus, setRegistryStatus] = useState("checking");
    const backdropClass = useMaybeBackdrop();
    const idPrefix = "network-configuration";

    const hasModal = backdropClass !== "";
    const isBlocked = hasActiveCheckpoint || hasModal;
    const registryReachable = registryStatus === "online";

    const checkRegistry = useCallback(() => {
        setRegistryStatus("checking");

        cockpit.spawn([
            "curl",
            "--silent",
            "--show-error",
            "--output", "/dev/null",
            "--write-out", "%{http_code}",
            "--connect-timeout", "5",
            "--max-time", "10",
            GHCR_CHECK_URL,
        ])
                .then(code => {
                    const status = code.trim();
                    setRegistryStatus(status === "200" || status === "401" ? "online" : "offline");
                })
                .catch(() => setRegistryStatus("offline"));
    }, []);

    useEffect(() => {
        if (!hasActiveCheckpoint) {
            checkRegistry();
        }
    }, [checkRegistry, hasActiveCheckpoint]);

    useEffect(() => {
        setIsFormValid(registryReachable && !isBlocked);
        setIsFormDisabled(isBlocked);
    }, [isBlocked, registryReachable, setIsFormDisabled, setIsFormValid]);

    let statusAlert;
    if (registryStatus === "checking") {
        statusAlert = (
            <Alert
              isInline
              isPlain
              title={_("Checking Internet connection")}
              variant="info">
                {_("JustVoxel is checking access to its GHCR installation source.")}
            </Alert>
        );
    } else if (registryReachable) {
        statusAlert = (
            <Alert
              isInline
              isPlain
              title={_("Internet connection ready")}
              variant="success">
                {_("GHCR is reachable. You can continue the installation.")}
            </Alert>
        );
    } else {
        statusAlert = (
            <Alert
              isInline
              title={_("Internet connection required")}
              variant="danger">
                {_("Configure Ethernet or Wi-Fi below. JustVoxel must reach GHCR before installation can continue.")}
                <div>
                    <Button
                      isInline
                      variant="link"
                      onClick={checkRegistry}>
                        {_("Check again")}
                    </Button>
                </div>
            </Alert>
        );
    }

    return (
        <div className={backdropClass + " " + idPrefix + "-page-section"}>
            {statusAlert}
            <CockpitNetworkIframe
              iframeId={idPrefix + "-frame"}
              iframeName="network-configuration"
              className={idPrefix + "-iframe"}
              onCritFail={onCritFail} />
        </div>
    );
};
