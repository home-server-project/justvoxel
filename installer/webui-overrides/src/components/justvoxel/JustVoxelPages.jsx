/*
 * Copyright (C) 2026 Home Server Project
 * SPDX-License-Identifier: LGPL-2.1-or-later
 */
import cockpit from "cockpit";

import React, { useEffect, useState } from "react";
import { Alert } from "@patternfly/react-core/dist/esm/components/Alert/index.js";
import { Button } from "@patternfly/react-core/dist/esm/components/Button/index.js";
import { Content } from "@patternfly/react-core/dist/esm/components/Content/index.js";
import { DescriptionList, DescriptionListDescription, DescriptionListGroup, DescriptionListTerm } from "@patternfly/react-core/dist/esm/components/DescriptionList/index.js";
import { Form, FormGroup } from "@patternfly/react-core/dist/esm/components/Form/index.js";
import { Radio } from "@patternfly/react-core/dist/esm/components/Radio/index.js";
import { Title } from "@patternfly/react-core/dist/esm/components/Title/index.js";
import { Stack } from "@patternfly/react-core/dist/esm/layouts/Stack/index.js";

import { checkJustVoxelEditionAvailable, setJustVoxelEdition } from "../../apis/justvoxel-bootc.js";

const _ = cockpit.gettext;

const useAlwaysValid = (setIsFormValid) => {
    useEffect(() => {
        setIsFormValid(true);
    }, [setIsFormValid]);
};

const EDITION_STORAGE_KEY = "justvoxel-edition";

const JustVoxelEdition = ({ idPrefix, setIsFormDisabled, setIsFormValid }) => {
    const savedEdition = window.sessionStorage.getItem(EDITION_STORAGE_KEY);
    const [edition, setEdition] = useState(savedEdition === "vm" ? "vm" : "hws");
    const [recommendedEdition, setRecommendedEdition] = useState("hws");
    const [detecting, setDetecting] = useState(true);
    const [applying, setApplying] = useState(false);
    const [configuredRef, setConfiguredRef] = useState("");
    const [configurationError, setConfigurationError] = useState("");
    const [retry, setRetry] = useState(0);

    useEffect(() => {
        let cancelled = false;

        cockpit.spawn(["systemd-detect-virt", "--vm"], { err: "message" })
                .then(output => {
                    if (cancelled) {
                        return;
                    }

                    const detected = output.trim();
                    const recommendation = detected && detected !== "none" ? "vm" : "hws";
                    setRecommendedEdition(recommendation);
                    if (!window.sessionStorage.getItem(EDITION_STORAGE_KEY)) {
                        setEdition(recommendation);
                    }
                })
                .catch(() => {
                    if (!cancelled) {
                        setRecommendedEdition("hws");
                        if (!window.sessionStorage.getItem(EDITION_STORAGE_KEY)) {
                            setEdition("hws");
                        }
                    }
                })
                .finally(() => {
                    if (!cancelled) {
                        setDetecting(false);
                    }
                });

        return () => {
            cancelled = true;
        };
    }, []);

    useEffect(() => {
        if (detecting) {
            setIsFormValid(false);
            setIsFormDisabled?.(true);
            return undefined;
        }

        let cancelled = false;
        setApplying(true);
        setConfigurationError("");
        setConfiguredRef("");
        setIsFormValid(false);
        setIsFormDisabled?.(true);

        checkJustVoxelEditionAvailable(edition)
                .then(() => setJustVoxelEdition(edition))
                .then(imageRef => {
                    if (!cancelled) {
                        window.sessionStorage.setItem(EDITION_STORAGE_KEY, edition);
                        setConfiguredRef(imageRef);
                        setIsFormValid(true);
                    }
                })
                .catch(error => {
                    if (!cancelled) {
                        setConfigurationError(error.message || String(error));
                        setIsFormValid(false);
                    }
                })
                .finally(() => {
                    if (!cancelled) {
                        setApplying(false);
                        setIsFormDisabled?.(false);
                    }
                });

        return () => {
            cancelled = true;
        };
    }, [detecting, edition, retry, setIsFormDisabled, setIsFormValid]);

    const disabled = detecting || applying;

    return (
        <Stack hasGutter>
            <Alert
              isInline
              isPlain
              title={detecting
                  ? _("Detecting installation environment")
                  : recommendedEdition === "vm"
                      ? _("Virtual machine detected")
                      : _("JustVoxel HWS is recommended")}
              variant="info">
                {!detecting && (recommendedEdition === "vm"
                    ? _("JustVoxel VM is selected by default. You can choose HWS instead.")
                    : _("No virtual machine was detected, so HWS is selected by default. You can choose VM instead."))}
            </Alert>

            <Form onSubmit={event => event.preventDefault()}>
                <FormGroup label={_("Choose your JustVoxel edition")} isStack>
                    <Radio
                      id={idPrefix + "-hws"}
                      name="justvoxel-edition"
                      label="JustVoxel HWS 10"
                      description={recommendedEdition === "hws"
                          ? _("Hardware Support edition. Recommended for this system.")
                          : _("Hardware Support edition.")}
                      isChecked={edition === "hws"}
                      isDisabled={disabled}
                      onChange={() => setEdition("hws")}
                    />
                    <Radio
                      id={idPrefix + "-vm"}
                      name="justvoxel-edition"
                      label="JustVoxel VM 10"
                      description={recommendedEdition === "vm"
                          ? _("Virtual machine edition. Recommended for this system.")
                          : _("Virtual machine edition.")}
                      isChecked={edition === "vm"}
                      isDisabled={disabled}
                      onChange={() => setEdition("vm")}
                    />
                </FormGroup>
            </Form>

            {applying &&
                <Alert isInline isPlain title={_("Checking selected edition")} variant="info">
                    {_("Verifying anonymous access to the selected JustVoxel image and updating the installation source.")}
                </Alert>}

            {configuredRef &&
                <Alert isInline isPlain title={_("Edition ready")} variant="success">
                    {cockpit.format(_("Installer source: $0"), configuredRef)}
                </Alert>}

            {configurationError &&
                <Alert isInline title={_("Selected JustVoxel edition is not available")} variant="danger">
                    {configurationError}
                    <div>
                        <Button isInline variant="link" onClick={() => setRetry(value => value + 1)}>
                            {_("Try again")}
                        </Button>
                    </div>
                </Alert>}
        </Stack>
    );
};

const LoginInformation = ({ setIsFormValid }) => {
    useAlwaysValid(setIsFormValid);

    // Informational only: interactive-defaults.ks owns all account settings.
    return (
        <Stack hasGutter>
            <Title headingLevel="h3">{_("Default administrator account")}</Title>
            <DescriptionList isHorizontal>
                <DescriptionListGroup>
                    <DescriptionListTerm>{_("Username")}</DescriptionListTerm>
                    <DescriptionListDescription>voxel</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>{_("Initial password")}</DescriptionListTerm>
                    <DescriptionListDescription>voxel</DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
            <Content>
                <Content component="p">{_("The password must be changed at first login.")}</Content>
                <Content component="p">{_("Root password login is disabled.")}</Content>
                <Content component="p">{_("These are development bootstrap credentials.")}</Content>
            </Content>
        </Stack>
    );
};

export class EditionPage {
    constructor () {
        this.component = JustVoxelEdition;
        this.id = "anaconda-screen-justvoxel-edition";
        this.label = _("JustVoxel edition");
        this.title = _("JustVoxel edition");
    }
}

export class LoginInformationPage {
    constructor () {
        this.component = LoginInformation;
        this.id = "anaconda-screen-justvoxel-login";
        this.label = _("Login information");
        this.title = _("Login information");
    }
}
