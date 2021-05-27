// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

import { By } from "selenium-webdriver";
import { Utils } from "./Utils";
import { Wait } from "./Wait";

export class Actions {
  // Scrolls viewport until element is present in viewport
  public static async scrollIntoView(
    by: By,
    viewportTop: boolean = true
  ): Promise<void> {
    const element = await Wait.waitForPresent(by);
    const driver = await Utils.getDriver();
    await driver.executeScript(
      `arguments[0].scrollIntoView(${viewportTop})`,
      element
    );
  }

  // Waits until the web element is visible and then clicks on it
  public static async doClick(by: By): Promise<void> {
    const element = await Wait.waitForEnable(by);
    console.log(`Clicking an element "${by}"`);
    await element.click();
  }

  // Wait until the web element is visible then enter the given text in HTMLInputElement using javascript executor
  public static async enterText(
    by: By,
    text: string,
    secret?: boolean
  ): Promise<void> {
    try {
      const e = await Wait.waitForVisible(by);
      console.log(`Entering text "${secret ? "********" : text}" in ${by}`);
      const driver = await Utils.getDriver();
      await driver.executeScript(
        `Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value").set.call(arguments[0], "${text}");` +
          `var event = document.createEvent('Event', {bubbles: true});` +
          `event.initEvent("input", true, true);` +
          `arguments[0].dispatchEvent(event)`,
        e
      );
    } catch (error) {
      console.error(`Could not enter text in ${by}`);
      throw error;
    }
  }
}
