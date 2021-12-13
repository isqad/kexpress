import React, { useEffect, useState } from 'react';
import {
  Link
} from "react-router-dom";

export default function Rubrics() {
  const [isLoading, setIsLoading] = useState(false);
  const [categories, setCategories] = useState([]);
  const [currenRubric, setCurrentRubric] = useState(null);

  useEffect(() => {
    if (isLoading) {
      return;
    }

    setIsLoading(true);

    fetch('/api/v1/roots', {
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json'
      }
    }).then(response => response.json())
      .then(categories => {
        setCategories(categories);
        //setIsLoading(false);
      });
  });

  return (
      <div>
        <h3>Категории</h3>
        <ul>
          {categories.map(category => {
            return (
              <li key={category.projectId}>
                {currenRubric === category.projectId
                  ? <small>{category.title} ({category.productAmount})</small>
                  : <small><Link to={`/rubrics/${category.projectId}`} onClick={() => setCurrentRubric(category.projectId)}>{category.title}</Link> ({category.productAmount})</small>
                }
              </li>
            );
          })}
        </ul>
      </div>
  );
}
