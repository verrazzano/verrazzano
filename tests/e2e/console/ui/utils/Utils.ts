// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

import * as fs from "fs";
import {
  Builder,
  Capabilities,
  Condition,
  WebDriver,
} from "selenium-webdriver";
import { MainPage } from "../pageObjects/MainPage.pom";
import { LoginPage } from "../pageObjects/LoginPage.pom";
export interface LoginInfo {
  username: string;
  password: string;
}

export class Utils {
  private static driver: WebDriver;
  private static config: any;

  static isLoginEnabled(): boolean {
    const rawLoginEnabled = Utils.getConfig("loginEnabled");
    // Anything other than the string or boolean false value is assumed to be a true
    if (rawLoginEnabled !== "false" && rawLoginEnabled !== false) {
      return true;
    } else {
      return false;
    }
  }

  static async navigateAndLogin(acceptCookies?: boolean, timeout?: number) {
    const url = Utils.getConfig("driverInfo").url as string;
    const loginInfo = Utils.getConfig("loginInfo");
    const loginEnabled = Utils.isLoginEnabled();
    try {
      console.log(`Navigating to: ${url}`);
      await Utils.getJETPage(url, timeout);

      if (loginEnabled) {
        Utils.validateConfigLoginInfo();
        const loginPage = new LoginPage();
        if (await loginPage.isPageLoaded()) {
          console.log("Login page is current page");
          console.log(`Performing an initial log in.`);
          await loginPage.login(loginInfo, acceptCookies, timeout);
        } else {
          const driver = await Utils.getDriver();
          console.log(
            `Login page is not current page. Current page url is ${await driver.getCurrentUrl()}`
          );
        }
      }
    } catch (error) {
      console.error(`Unable to navigate and log in to ${url}. ${error}`);
      throw error;
    }
  }

  // pageReadyScript is the Javascript code for Oracle JET page ready check
  static pageReadyScript = `
    const done = arguments[0];
    const contextModule = 'ojs/ojcontext';
    try {
      require(
        [contextModule],
        function(Context) {
          if (Context.getPageContext().getBusyContext().isReady()) {
            done('');
          } else {
            done('not yet ready');
          }
        },
        function (ex) {
          require.undef(contextModule);
          done(ex.message);
        }
      )
    } catch (ex) {
      if (ex.message === 'require is not defined') {
         done(''); // Not a JET page
      } else {
         done(ex.message);
      } 
    }
    `;

  // ojetPageReady() returns a Condition that executes the page ready Javascript check remotely on the browser
  static ojetPageReady(): Condition<Promise<boolean>> {
    return new Condition("Page Ready", async (driver: WebDriver) => {
      try {
        console.debug("Running page ready script:");
        const scriptOutput = await driver.executeAsyncScript<string>(
          Utils.pageReadyScript
        );
        console.debug(
          `Ran page ready script - return value: "${scriptOutput}"`
        );
        return scriptOutput === "";
      } catch (err) {
        return false;
      }
    });
  }

  static getConfig(key: string) {
    if (!Utils.config) {
      Utils.readConfigFile();
    }
    return key ? Utils.config[key] : Utils.config;
  }

  static async gotoMainPage(): Promise<MainPage> {
    const mainPage = new MainPage();
    const uiUrl = Utils.getConfig("driverInfo").url;
    console.log(`Navigating to UI main page at ${uiUrl}`);
    await Utils.getJETPage(uiUrl);

    // Verify MainPage is reachable and loaded
    await mainPage.isPageLoaded();
    return mainPage;
  }

  public static releaseDriver() {
    if (Utils.driver) {
      setTimeout(async () => {
        try {
          await Utils.driver.quit();
        } catch (err) {
          console.warn(`Failed when releasing driver session: ${err}`);
        } finally {
          Utils.driver = null;
        }
      }, 500);
    }
  }

  public static async getDriver(): Promise<WebDriver> {
    if (!Utils.driver) {
      const driverInfo = Utils.getConfig("driverInfo");

      const caps = new Capabilities(driverInfo);
      const driver: WebDriver = new Builder().withCapabilities(caps).build();
      Utils.driver = driver;
    }
    return Utils.driver;
  }

  static readConfigFile(): void {
    const configPath = process.env.VZ_UITEST_CONFIG || "config.uitest.json";
    console.log(`Reading UI test config file at ${configPath}`);
    if (fs.existsSync(configPath)) {
      if (configPath.endsWith(".json")) {
        Utils.config = JSON.parse(fs.readFileSync(configPath).toString());
      } else {
        throw new Error(
          `The uitest config file ${configPath}, must have a name ending in ".json"`
        );
      }
    } else {
      throw new Error(
        `No uitest config file found for file name ${configPath}.`
      );
    }

    if (!Utils.config.driverInfo) {
      Utils.config.driverInfo = { url: "http://localhost:8000" };
    } else if (!Utils.config.driverInfo.url) {
      Utils.config.driverInfo.url = "http://localhost:8000";
    }
  }

  private static validateConfigLoginInfo() {
    const errMsg = `The uitest config file must have a loginInfo section with username and password`;
    if (!Utils.config.loginInfo) {
      throw new Error(errMsg);
    }
    const loginInfo = Utils.config.loginInfo as LoginInfo;
    if (!loginInfo.username || !loginInfo.password) {
      throw new Error(errMsg);
    }
  }

  private static async getJETPage(url: string, timeout?: number) {
    const driver = await Utils.getDriver();
    await driver.get(url);
    await driver.wait(Utils.ojetPageReady(), timeout);
  }
}
