// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

import { By } from "selenium-webdriver";
import { LoginInfo } from "../utils/Utils";
import { Wait, PAGE_LOAD_TIMEOUT } from "../utils/Wait";
import { Actions } from "../utils/Actions";

/**
 * Page Object Model for the Keycloak login page
 */
export class LoginPage {
  private static readonly LOGIN_FORM_BY: By = By.id("kc-form-login");
  private static readonly USERNAME_BY: By = By.id("username");
  private static readonly PASSWORD_BY: By = By.id("password");
  private static readonly LOGIN_BTN_BY: By = By.id("kc-login");
  protected pageUrl: string = "/";
  protected pageLoadedElement: By = LoginPage.LOGIN_FORM_BY;

  public async isCurrentPage(): Promise<boolean> {
    const elem = await Wait.waitForPresent(LoginPage.LOGIN_FORM_BY);
    return !!elem;
  }

  public async isPageLoaded(
    timeOut: number = PAGE_LOAD_TIMEOUT
  ): Promise<boolean> {
    try {
      await Wait.waitForPresent(this.pageLoadedElement, timeOut);
      return true;
    } catch (error) {
      return false;
    }
  }

  public async login(
    loginInfo: LoginInfo,
    acceptCookies?: boolean,
    timeout?: number
  ) {
    console.log("Performing Keycloak Login");
    const isUsernameBoxPresent = await Wait.waitForPresent(
      LoginPage.USERNAME_BY
    )
      .then(() => true)
      .catch(() => false);

    if (isUsernameBoxPresent) {
      await Actions.enterText(LoginPage.USERNAME_BY, loginInfo.username);
      await Actions.enterText(LoginPage.PASSWORD_BY, loginInfo.password, true);
      await Actions.doClick(LoginPage.LOGIN_BTN_BY);
    } else {
      throw new Error("No username box, could not login");
    }
  }
}
