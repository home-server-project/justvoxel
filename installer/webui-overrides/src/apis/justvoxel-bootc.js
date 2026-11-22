/*
 * Copyright (C) 2026 Home Server Project
 * SPDX-License-Identifier: LGPL-2.1-or-later
 */
import cockpit from "cockpit";

import { getActivePayload, PayloadsClient } from "./payloads.js";

const DBUS_PROPERTIES = "org.freedesktop.DBus.Properties";
const PAYLOAD_INTERFACE = "org.fedoraproject.Anaconda.Modules.Payloads.Payload";
const SOURCE_INTERFACE = "org.fedoraproject.Anaconda.Modules.Payloads.Source";
const BOOTC_INTERFACE = "org.fedoraproject.Anaconda.Modules.Payloads.Source.Bootc";

export const JUSTVOXEL_IMAGE_REFS = {
    hws: "registry:ghcr.io/home-server-project/justvoxel-hws:stable",
    vm: "registry:ghcr.io/home-server-project/justvoxel-vm:stable",
};

const JUSTVOXEL_REGISTRY_REPOSITORIES = {
    hws: "home-server-project/justvoxel-hws",
    vm: "home-server-project/justvoxel-vm",
};

const MANIFEST_ACCEPT = [
    "application/vnd.oci.image.index.v1+json",
    "application/vnd.oci.image.manifest.v1+json",
    "application/vnd.docker.distribution.manifest.list.v2+json",
    "application/vnd.docker.distribution.manifest.v2+json",
].join(", ");

const getClient = () => {
    const client = PayloadsClient.instance?.client;
    if (!client) {
        throw new Error("Anaconda payload service is not available");
    }
    return client;
};

const getProperty = async (client, objectPath, interfaceName, propertyName) => {
    const result = await client.call(
        objectPath,
        DBUS_PROPERTIES,
        "Get",
        [interfaceName, propertyName]
    );
    return result[0].v;
};

const findBootcSource = async (client) => {
    const activePayload = await getActivePayload();
    const sources = await getProperty(client, activePayload, PAYLOAD_INTERFACE, "Sources");

    for (const source of sources) {
        const sourceType = await getProperty(client, source, SOURCE_INTERFACE, "Type");
        if (sourceType === "BOOTC") {
            return source;
        }
    }

    throw new Error("Active Anaconda payload has no BOOTC source");
};

export const checkJustVoxelEditionAvailable = async (edition) => {
    const repository = JUSTVOXEL_REGISTRY_REPOSITORIES[edition];
    if (!repository) {
        throw new Error("Unknown JustVoxel edition");
    }

    const tokenUrl = "https://ghcr.io/token?service=ghcr.io&scope=" +
        encodeURIComponent("repository:" + repository + ":pull");

    let token;
    try {
        const tokenResponse = await cockpit.spawn([
            "curl",
            "--silent",
            "--show-error",
            "--fail",
            "--connect-timeout", "5",
            "--max-time", "10",
            tokenUrl,
        ], { err: "message" });
        token = JSON.parse(tokenResponse).token;
    } catch (error) {
        throw new Error("Unable to get anonymous GHCR pull access for the selected image");
    }

    if (!token) {
        throw new Error("Selected JustVoxel image is not available for anonymous pull");
    }

    let status;
    try {
        status = await cockpit.spawn([
            "curl",
            "--silent",
            "--show-error",
            "--output", "/dev/null",
            "--write-out", "%{http_code}",
            "--head",
            "--connect-timeout", "5",
            "--max-time", "10",
            "--header", "Authorization: Bearer " + token,
            "--header", "Accept: " + MANIFEST_ACCEPT,
            "https://ghcr.io/v2/" + repository + "/manifests/stable",
        ], { err: "message" });
    } catch (error) {
        throw new Error("Unable to reach the selected JustVoxel image on GHCR");
    }

    const httpStatus = status.trim();
    if (httpStatus === "200") {
        return true;
    }
    if (httpStatus === "401" || httpStatus === "403") {
        throw new Error("Selected JustVoxel image is not public");
    }
    if (httpStatus === "404") {
        throw new Error("Selected JustVoxel stable image was not found");
    }

    throw new Error("Selected JustVoxel image is unavailable on GHCR");
};

export const setJustVoxelEdition = async (edition) => {
    const imageRef = JUSTVOXEL_IMAGE_REFS[edition];
    if (!imageRef) {
        throw new Error("Unknown JustVoxel edition");
    }

    const client = getClient();
    const bootcSource = await findBootcSource(client);
    const current = await getProperty(client, bootcSource, BOOTC_INTERFACE, "Configuration");

    const configuration = {
        stateroot: cockpit.variant("s", current.stateroot?.v || ""),
        sourceImgRef: cockpit.variant("s", imageRef),
        targetImgRef: cockpit.variant("s", imageRef),
    };

    await client.call(
        bootcSource,
        DBUS_PROPERTIES,
        "Set",
        [
            BOOTC_INTERFACE,
            "Configuration",
            cockpit.variant("a{sv}", configuration),
        ]
    );

    return imageRef;
};
