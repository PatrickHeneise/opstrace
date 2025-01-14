/**
 * Copyright 2020 Opstrace, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import express from "express";

import { GeneralServerError } from "server/errors";
import datasourceHandler from "./datasource";
import createAuthHandler from "./authentication";
import createGraphqlHandler from "./graphql";
import pubUiCfgHandler from "./uicfg";

function api(): express.Router {
  const api = express.Router();
  api.use("/auth", createAuthHandler());
  api.use("/public-ui-config", pubUiCfgHandler);
  api.use("/graphql", createGraphqlHandler());
  api.use("/datasource/:target", datasourceHandler);
  api.all("*", function (req, res, next) {
    next(new GeneralServerError(404, "api route not found"));
  });

  return api;
}

export default api;
