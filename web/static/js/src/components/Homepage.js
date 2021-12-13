import React, { useEffect, useState } from 'react';
import Rubrics from './Rubrics.js';
import RubricStatistics from './RubricStatistics.js';
import {
  BrowserRouter as Router,
  Switch,
  Route
} from "react-router-dom";

export default function Homepage() {
  return (
    <Router>
      <div className="container">
        <div className="row">
          <div className="col-3">
            <Rubrics />
          </div>
          <div className="col-9">
            <Switch>
              <Route path="/rubrics/:id" children={<RubricStatistics />} />
            </Switch>
          </div>
        </div>
      </div>
    </Router>
  );
}
