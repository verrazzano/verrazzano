// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

import { MainPage } from "./pageObjects/MainPage.pom";
import { HeaderBar } from "./pageObjects/HeaderBar.pom";
import { expect } from "chai";
import { Utils } from "./utils/Utils";

describe("Instance Details Page", (): void => {
  let mainPage: MainPage;
  let headerBar: HeaderBar;

  before(async () => {
    await Utils.navigateAndLogin();
    headerBar = new HeaderBar();
    mainPage = await Utils.gotoMainPage();
  });

  describe("Access Header and footer", (): void => {
    it("Main Page should load and contain header", async () => {
      expect(await mainPage.waitForHeader()).to.be.true;
    });

    it("Main Page should contain footer", async () => {
      expect(await mainPage.waitForFooter()).to.be.true;
    });
  });

  describe("Access Header logo", (): void => {
    it("Wait for header", async () => {
      await mainPage.waitForHeader();
    });

    it("Access header logo", async () => {
      expect(await headerBar.selectLogo()).to.be.true;
    });
  });

  after(() => {
    Utils.releaseDriver();
  });
});
